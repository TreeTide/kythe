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
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
  "strconv"

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

  "github.com/golang/protobuf/proto"
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

type BatchState struct {
  batchCounter int
  hasher hash.Hash64
  dbConn *pgx.Conn
  dbBatch *pgx.Batch
  dbBatchCount int
  batch []*(spb.Entry)
  kind scpb.NodeKind
  startByte int
  endByte int
  childofSigl uint64
}

func (s *BatchState) processEntry(entry *spb.Entry) error {
  if len(s.batch) == 0 || proto.Equal(s.batch[0].Source, entry.Source) {
    switch entry.FactName {
    case facts.NodeKind:
      s.kind = schema.NodeKind(string(entry.FactValue))
    case facts.AnchorStart:
      if b, err := strconv.Atoi(string(entry.FactValue)); err == nil {
        s.startByte = b
      }
    case facts.AnchorEnd:
      if b, err := strconv.Atoi(string(entry.FactValue)); err == nil {
        s.endByte = b
      }
    default:
      // eh
    }

    switch entry.EdgeKind {
    case "":
    case edges.ChildOf:
      s.childofSigl = s.siglHash(entry.Target)
    default:
      s.batch = append(s.batch, entry)
    }
  } else {
    s.finalizeBatch()
    return s.processEntry(entry)
  }
  return nil;
}

func (s *BatchState) crpHash(v *spb.VName) uint64 {
  s.hasher.Reset()
  s.hasher.Write([]byte(v.Corpus))
  s.hasher.Write([]byte(v.Root))
  s.hasher.Write([]byte(v.Path))
  return s.hasher.Sum64() >> 1
}

func (s *BatchState) siglHash(v *spb.VName) uint64 {
  s.hasher.Reset()
  s.hasher.Write([]byte(v.Signature))
  s.hasher.Write([]byte(v.Language))
  return s.hasher.Sum64() >> 1
}

func (s *BatchState) finalizeBatch() {
  rep := s.batch[0]
  src := rep.Source
  pathHash := s.crpHash(src)
  sigHash := s.siglHash(src)

  s.batchCounter++
  shouldRecord := s.batchCounter % 100 == 0

  if shouldRecord {
    // subsampled
    s.dbBatch.Queue("INSERT INTO crp (crp, corpus, root, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING", pathHash, rep.Source.Corpus, rep.Source.Root, rep.Source.Path)

  /*
  fmt.Printf("==== Batch for %v (%v : %v) ====\n", src, pathHash, sigHash)
  fmt.Printf("%v %v %v %v\n", s.kind, s.startByte, s.endByte, s.childofSigl)
  */

    if s.kind == scpb.NodeKind_ANCHOR {
      for _, e := range s.batch {
        ekind := schema.EdgeKind(string(e.EdgeKind))
        tcrp := s.crpHash(e.Target)
        tsigl := s.siglHash(e.Target)
        //fmt.Printf("%v %v %v\n", ekind, s.crpHash(e.Target), s.siglHash(e.Target))

        /*
         CREATE TABLE anchor (crp bigint, sigl bigint, psigl bigint, bs int, be int, ekind int, tcrp bigint, tsigl bigint);

         CREATE TABLE crp (crp BIGINT, corpus TEXT, root TEXT, path TEXT);
        */
          // so so we only get crp assignment
        s.dbBatch.Queue("INSERT INTO anchor (crp, sigl, psigl, bs, be, ekind, tcrp, tsigl) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)", pathHash, sigHash, s.childofSigl, s.startByte, s.endByte, ekind, tcrp, tsigl)

        s.dbBatchCount += 1
        s.insertBatch(false)
      }
      // TODO collect: ref, defines/binding, childof
    }

  }

  /*
  for _, e := range s.batch {
    fmt.Printf("- %v\n", e.String())
  }
  */
  s.batch = s.batch[:0]
  s.kind = scpb.NodeKind_UNKNOWN_NODE_KIND
  s.startByte = -1
  s.endByte = -1
  s.childofSigl = 0
}

func (s *BatchState) insertBatch(force bool) {
  if (force || s.dbBatchCount >= 500) {
    dbmux.Lock()
    defer dbmux.Unlock()  // TODO use chans etc
    br := s.dbConn.SendBatch(context.Background(), s.dbBatch)
    _, err := br.Exec()
    if err != nil {
      panic(err)
    }
    s.dbBatchCount = 0
    err = br.Close()
    if err != nil {
      panic(err)
    }
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

  batch := BatchState {
    dbConn: conn,
    dbBatch: &pgx.Batch{},
    hasher: fnv.New64a(),
    batch: make([]*spb.Entry, 0, 100),
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
