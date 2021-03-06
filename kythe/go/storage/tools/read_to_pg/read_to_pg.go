/*
 * Copyright 2014 The Kythe Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Binary read_to_pg scans the entries from a specified GraphStore and emits
// them to a PostgreSQL database for further processing.
//
//     CREATE TABLE anchor0 (crp bigint NOT NULL, sigl bigint NOT NULL, bs int NOT NULL, be int NOT NULL, ekind int NOT NULL, tcrp bigint NOT NULL, tsigl bigint NOT NULL);
//
//      CREATE INDEX a0_crp ON anchor0(crp);
//      CREATE INDEX a0_target ON anchor0(tsigl, tcrp);
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
  "strconv"

  "database/sql"

	"kythe.io/kythe/go/services/graphstore"
	"kythe.io/kythe/go/storage/gsutil"
	"kythe.io/kythe/go/util/flagutil"
	"kythe.io/kythe/go/util/kytheuri"
	"kythe.io/kythe/go/util/schema"
	"kythe.io/kythe/go/util/schema/facts"
	"kythe.io/kythe/go/util/schema/edges"

  scpb "kythe.io/kythe/proto/schema_go_proto"
	spb "kythe.io/kythe/proto/storage_go_proto"

	_ "kythe.io/kythe/go/services/graphstore/proxy"
	_ "kythe.io/kythe/go/storage/leveldb"

  //"github.com/golang/protobuf/proto"
  "github.com/jackc/pgx/v4"

  "hash"
  "hash/fnv"

  "sync"
)

var (
	gs graphstore.Service

	edgeKind     = flag.String("edge_kind", "", "Edge kind by which to filter a read/scan")
	targetTicket = flag.String("target", "", "Ticket of target by which to filter a scan")
	factPrefix   = flag.String("fact_prefix", "", "Fact prefix by which to filter a scan")
)

func init() {
	gsutil.Flag(&gs, "graphstore", "GraphStore to read")
	flag.Usage = flagutil.SimpleUsage("Scans/reads the entries from a GraphStore, writing the content to the given PostgreSQL database.",
		"--graphstore spec [--edge_kind] ([--fact_prefix str] [--target ticket] | [ticket...])")
}

var dbmux sync.Mutex

const kTupleBatchMax int = 100000

type BatchState struct {
  batchCounter int
  hasher hash.Hash64
  dbConn *pgx.Conn
  dbBatch *pgx.Batch
  dbBatchCount int

  seenCrp map[int64]bool
  seenSigl map[int64]bool

  anchorEdgeBatch [][]interface{}
  anchorBatch [][]interface{}

  // The crp of the last entry processed. Only used to check
  latestCrp int64
  // The sigl of the first entry in the batch.
  sigl int64

  // Facts collected while scanning entries of a single sigl.
  kind scpb.NodeKind
  // For kind=anchor
  // (Note: anchors can be incident, so sigl is needed to disambiguate them)
  startByte sql.NullInt32
  endByte sql.NullInt32

  // The entries collected for a given sigl.
  batch []*(spb.Entry)

  // Accumulates stuff for a given crp.
  // For kind=anchor
  sigls []int64
  ekinds []int32
  tcrps []int64
  tsigls []int64
  bss []int
  bes []int
  //childofSigl uint64
}

func initDb(conn *pgx.Conn) error {
  _, err := conn.Exec(context.Background(), "DROP TABLE anchor")
  if err != nil {
    log.Printf("Couldn't DROP anchor table: %v", err)
  }
  conn.Exec(context.Background(), "DROP TABLE anchor_edge")
  conn.Exec(context.Background(), "DROP TABLE crp")
  if _, err := conn.Exec(context.Background(),
      "CREATE TABLE anchor (crp bigint NOT NULL, sigl bigint NOT NULL, bs int, be int)"); err != nil { return err }
  if _, err := conn.Exec(context.Background(),
      "CREATE TABLE anchor_edge (crp bigint NOT NULL, sigl bigint NOT NULL, ekind int NOT NULL, tcrp bigint NOT NULL, tsigl bigint NOT NULL)"); err != nil { return err }

  if _, err := conn.Exec(context.Background(),
      "CREATE TABLE crp (crp bigint NOT NULL, corpus TEXT, root TEXT, path TEXT)"); err != nil { return err }
  return nil
  /*
  CREATE TABLE crp (crp BIGINT NOT NULL, corpus TEXT, root TEXT, path TEXT);
  */
}

func (s* BatchState) maybeRecordCrpSigl(vname *spb.VName, crp int64, sigl int64) {
  _, hasCrp := s.seenCrp[crp]
  if !hasCrp {
    s.seenCrp[crp] = true
    s.dbBatch.Queue(
      "INSERT INTO crp (crp, corpus, root, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
      crp, vname.Corpus, vname.Root, vname.Path)
    s.dbBatchCount++
  }

  /* Seeing sing has not much value, except the language (which could be added separately?).
  _, hasSigl := s.seenSigl[sigl]
  if !hasSigl {
    s.seenSigl[sigl] = true
    s.dbBatch.Queue(
      "INSERT INTO sigl (sigl, signature, language) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
      sigl, vname.Signature, vname.Language)
    s.dbBatchCount++
  }
  */

  s.maybeFlush(false)
}

func (s* BatchState) maybeFlush(force bool) {
  if force || s.dbBatchCount > 100 {
    log.Printf("Flushing batch of crp&sigl")
    br := s.dbConn.SendBatch(context.Background(), s.dbBatch)
    _, err := br.Exec()
    if err != nil {
      panic(err)
    }
    err = br.Close()
    if err != nil {
      panic(err)
    }
    s.dbBatch = &pgx.Batch{}
    s.dbBatchCount = 0
  }
}

func (s *BatchState) processEntry(entry *spb.Entry) error {
  // To serialize access to DB / fields. Is processEntry called serially?
  //log.Printf("processEntry")
  dbmux.Lock()

  crp := s.crpHash(entry.Source)
  sigl := s.siglHash(entry.Source)

  s.maybeRecordCrpSigl(entry.Source, crp, sigl)

  //log.Printf("CRP %v %v %v", entry.Source.Corpus, entry.Source.Root, entry.Source.Path)
  batchLen := len(s.batch)
  if batchLen == 0 || (s.sigl == sigl && s.latestCrp == crp) {
    if batchLen == 0 {
      s.sigl = sigl
      s.latestCrp = crp
      //log.Printf("Processing new sig %v (%v)", entry.Source, sigl)
    }
    switch entry.FactName {
    case facts.NodeKind:
      s.kind = schema.NodeKind(string(entry.FactValue))
      //log.Printf("recording node kind %v", s.kind)
    case facts.AnchorStart:
      if b, err := strconv.Atoi(string(entry.FactValue)); err == nil {
        s.startByte.Int32 = int32(b)
        s.startByte.Valid = true
      }
    case facts.AnchorEnd:
      if b, err := strconv.Atoi(string(entry.FactValue)); err == nil {
        s.endByte.Int32 = int32(b)
        s.endByte.Valid = true
      }
    // Note: snippet bounds are not stored in graphstore.
    default:
      // eh
    }

    switch entry.EdgeKind {
    case "":
    case edges.ChildOf:
      //s.childofSigl = s.siglHash(entry.Target)
    default:
      s.batch = append(s.batch, entry)
    }
  } else {
    // Entries of a new entity begin, so finish extracting info from current
    // batch.
    s.finalizeBatch()
    dbmux.Unlock()
    return s.processEntry(entry)
  }
  dbmux.Unlock()
  return nil;
}

func (s *BatchState) crpHash(v *spb.VName) int64 {
  s.hasher.Reset()
  s.hasher.Write([]byte(v.Corpus))
  s.hasher.Write([]byte(v.Root))
  s.hasher.Write([]byte(v.Path))
  // Shifting probably to make output fit postgres's int8 type?
  return int64(s.hasher.Sum64() >> 1)
}

func (s *BatchState) siglHash(v *spb.VName) int64 {
  s.hasher.Reset()
  s.hasher.Write([]byte(v.Signature))
  s.hasher.Write([]byte(v.Language))
  return int64(s.hasher.Sum64() >> 1)
}

// Accumulates crp+sigl-level data into the arrays.
func (s *BatchState) finalizeBatch() {
  //log.Printf("finalizeBatch")
  //rep := s.batch[0]
  //src := rep.Source

  //log.Printf("differing path hash")
  //log.Printf("sigls: %v for crp %v (%v : %v)", len(s.sigls), src, pathHash, sigHash)

  // TODO: local LRU cache for crp, only insert if not found in cache
  //
  //s.dbBatch.Queue(
  //  "INSERT INTO crp (crp, corpus, root, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
  //  pathHash, rep.Source.Corpus, rep.Source.Root, rep.Source.Path)
  s.insertBatch(false)

  s.batchCounter++
  shouldRecord := true //s.batchCounter % 100 < 1

  if shouldRecord {
    //log.Printf("shouldRecord")
    //fmt.Printf("%v %v %v %v\n", s.kind, s.startByte, s.endByte, s.childofSigl)

      s.anchorBatch = append(s.anchorBatch,
        []interface{}{int64(s.latestCrp), int64(s.sigl), s.startByte, s.endByte})

    //if s.kind == scpb.NodeKind_ANCHOR {
      for _, e := range s.batch {
        ekind := schema.EdgeKind(string(e.EdgeKind))
        if ekind == 0 {
          // Dotted param edge - skip for now. Might check to be sure.
          continue
        }
        tcrp := s.crpHash(e.Target)
        tsigl := s.siglHash(e.Target)

        s.maybeRecordCrpSigl(e.Target, tcrp, tsigl)

        s.anchorEdgeBatch = append(s.anchorEdgeBatch,
          []interface{}{int64(s.latestCrp), int64(s.sigl), int32(ekind), int64(tcrp), int64(tsigl)})
      }
    //}
  }

  // Reset sigl accum
  s.batch = s.batch[:0]  // TODO rename siglbatch
  s.sigl = -1
  s.latestCrp = -1
  s.kind = scpb.NodeKind_UNKNOWN_NODE_KIND
  s.startByte.Valid = false
  s.endByte.Valid = false
  //s.childofSigl = 0
}

// Sends current DB batch if forced or certain thresholds are reached.
// Note: could make async?
func (s *BatchState) insertBatch(force bool) {
  if (force || len(s.anchorEdgeBatch) >= kTupleBatchMax) {
    copyCount, err := s.dbConn.CopyFrom(context.Background(),
      []string{"anchor_edge"},
      []string{"crp", "sigl",  "ekind", "tcrp", "tsigl"},
      pgx.CopyFromRows(s.anchorEdgeBatch))
    if err != nil {
      panic(err)
    }
    if copyCount != int64(len(s.anchorEdgeBatch)) {
      log.Fatalf("Written count didn't match expected")
    }
    log.Printf("Copied %v anchor_edge rows", copyCount)
    s.anchorEdgeBatch = s.anchorEdgeBatch[:0]
  }

  if (force || len(s.anchorBatch) >= kTupleBatchMax) {
    copyCount, err := s.dbConn.CopyFrom(context.Background(),
      []string{"anchor"},
      []string{"crp", "sigl", "bs", "be"},
      pgx.CopyFromRows(s.anchorBatch))
    if err != nil {
      panic(err)
    }
    if copyCount != int64(len(s.anchorBatch)) {
      log.Fatalf("Written count didn't match expected")
    }
    log.Printf("Copied %v anchor rows", copyCount)
    s.anchorBatch = s.anchorBatch[:0]
  }

}

func main() {
	flag.Parse()
	if gs == nil {
		flagutil.UsageError("missing --graphstore")
	}
	ctx := context.Background()

  conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
  if err != nil {
    fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
    os.Exit(1)
  }
  defer conn.Close(ctx)

  log.Printf("Initing DB from scratch")
  if err := initDb(conn); err != nil {
    panic(err)
  }

  batch := BatchState {
    dbConn: conn,
    dbBatch: &pgx.Batch{},
    hasher: fnv.New64a(),
    batch: make([]*spb.Entry, 0, 100),
    anchorEdgeBatch: make([][]interface{}, 0, kTupleBatchMax),
    anchorBatch: make([][]interface{}, 0, kTupleBatchMax),
    seenCrp: make(map[int64]bool),
    seenSigl: make(map[int64]bool),
    startByte: sql.NullInt32{Int32: 0, Valid: false},
    endByte: sql.NullInt32{Int32: 0, Valid: false},
  }
  if len(flag.Args()) > 0 {
    if *targetTicket != "" || *factPrefix != "" {
      log.Fatal("--target and --fact_prefix are unsupported when given tickets")
    }
    if err := readEntries(ctx, gs, batch.processEntry, *edgeKind, flag.Args()); err != nil {
      log.Fatal(err)
    }
  } else {
    if err := scanEntries(ctx, gs, batch.processEntry, *edgeKind, *targetTicket, *factPrefix); err != nil {
      log.Fatal(err)
    }
  }
  batch.finalizeBatch()
  batch.insertBatch(true)
  batch.maybeFlush(true)
}

func readEntries(ctx context.Context, gs graphstore.Service, entryFunc graphstore.EntryFunc, edgeKind string, tickets []string) error {
	for _, ticket := range tickets {
		src, err := kytheuri.ToVName(ticket)
		if err != nil {
			return fmt.Errorf("error parsing ticket %q: %v", ticket, err)
		}
		if err := gs.Read(ctx, &spb.ReadRequest{
			Source:   src,
			EdgeKind: edgeKind,
		}, entryFunc); err != nil {
			return fmt.Errorf("GraphStore Read error for ticket %q: %v", ticket, err)
		}
	}
	return nil
}

func scanEntries(ctx context.Context, gs graphstore.Service, entryFunc graphstore.EntryFunc, edgeKind, targetTicket, factPrefix string) error {
	var target *spb.VName
	var err error
	if targetTicket != "" {
		target, err = kytheuri.ToVName(targetTicket)
		if err != nil {
			return fmt.Errorf("error parsing --target %q: %v", targetTicket, err)
		}
	}
	if err := gs.Scan(ctx, &spb.ScanRequest{
		EdgeKind:   edgeKind,
		FactPrefix: factPrefix,
		Target:     target,
	}, entryFunc); err != nil {
		return fmt.Errorf("GraphStore Scan error: %v", err)
	}
	return nil
}

