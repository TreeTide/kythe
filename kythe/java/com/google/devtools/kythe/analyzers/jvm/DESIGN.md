# Kythe JVM language

## Background

The JVM hosts many higher-level languages and allows for cross-language usage of
any class defined in any higher-level language.  It is desirable for Kythe to
model these cross-language references to support multi-language projects.

As of today (January 2018), among the languages that target the JVM, only Java
has a Kythe indexer.  Like most Kythe indexers, it mostly emits nodes with
opaque VNames only understood by the indexer emitting them.  This is an obstacle
for any other JVM-based language indexer that references Java classes since it
cannot construct edges into the Java semantic graph.  It is either necessary to
make the Java node names constructible and/or add a layer of indirection in the
Kythe graph.

The Kythe Protocol Buffer indexer shares a similar problem.  Many different
languages refer to entities defined by `.proto` source files and Kythe would
like to model cross-references across all of the protobuf target languages.
Kythe achieves this through a mix of utilizing metadata while writing generated
source code for each language in order to properly construct protobuf Kythe node
names and emitting `generates` edges to the generated source code from the
originating `.proto` source code.  The `generates` edges tie together all
cross-language references through the defining Kythe proto node as a **cluster**
of nodes sharing the protobuf node definition site.

This design proposes to use a similar mechanism, viewing the JVM `.class` files
generated by the Java compiler as a separate language and adding `generates`
edges from the existing Java semantic nodes to a new set of JVM semantic nodes.
By design, the JVM semantic node names will be constructible by indexers from
other JVM-based languages and can be targets of references in those languages.
The shared JVM nodes will allow the higher-level semantic nodes to be clustered
and share a definition site in the appropriate higher-level language.

Related work:

- https://phabricator-dot-kythe-repo.appspot.com/D1426:
    - This proposal creates a new JVM language graph primarily as an extension
    instead of a partial replacement of the existing Java graph as in D1426
- https://phabricator-dot-kythe-repo.appspot.com/D1959:
    - This proposal replaces the current `name` nodes as defined in D1959 with
    the new JVM language nodes

## Kythe schema

#### VNames

Unlike VNames from most Kythe-supported languages, JVM VNames are constructible
outside of the indexer that emits them since they exist to tie multiple
languages together.  Each JVM VName will have its language component set to
"jvm" and have a well-defined signature corresponding to a JVM entity using the
entity's qualified named (and possibly a differentiating JVM type descriptor).
These VName signatures are unrelated to the JVM concept of a signature.  All
other VName fields (i.e.  corpus/root/path) are empty.  See below for
descriptions of each signature per node kind.

```
signature: <qualified_name> + <type_descriptor>?
language: "jvm"
corpus: <empty>
root: <empty>
path: <empty>
```

It is left to future iterations to use the corpus/root/path components to
possibly differentiate between same-named JVM entities generated across corpora
(or Java modules).

#### JVM nodes

The following table describes the Kythe node corresponding to each major JVM
entity modeled.  Included are the node's kind, subkind, and VName signature.

| JVM Entity | `node/kind` | `subkind`   | VName signature                        | Example signature                               |
| ---        | ---         | ---         | ---                                    | ---                                             |
| Class      | `record`    | `class`     | `<qualified name>`                     | `java.util.ArrayList`                           |
| Interface  | `interface` |             | `<qualified name>`                     | `java.util.Map.Entry`                           |
| Enum       | `sum`       | `enumClass` | `<qualified name>`                     | `java.time.DayOfWeek`                           |
| Method     | `function`  |             | `<qualified name> + <type descriptor>` | `java.util.ArrayList.add(ILjava/lang/Object;)V` |
| Field      | `variable`  | `field`     | `<qualified name>`                     | `java.lang.String.CASE_INSENSITIVE_ORDER`       |

JVM type descriptors are generated as documented by
[JVMS §4.3](https://docs.oracle.com/javase/specs/jvms/se9/html/jvms-4.html#jvms-4.3).
All Java type variables (a.k.a. generics) are referenced as `Ljava.lang.Object;`
in the JVM language type descriptors.

<!-- TODO(schroederc): other nodes (e.g. parameters, locals, and types) -->

#### JVM edges

Initially, the JVM language will only have edges describing the hierarchy of
classes and their members.

```
class member -[childof]-> class
```

<!-- TODO(schroederc): other edges (e.g. types, parameters, locals, callgraph) -->

#### Relations with higher-level JVM languages

For each **definition** in a source file of a higher-level language targeting
the JVM (such as Java, Scala, Clojure, Kotlin, etc.), a Kythe `generates` edge
should exist between the higher-level semantic node and the corresponding JVM
semantic node (i.e. a Java class definition generates the JVM class node, a Java
method definition generates the JVM method node, etc.).

```
Java node    -[generates]-> JVM node
            .. OR ..
Scala node   -[generates]-> JVM node
            .. OR ..
Clojure node -[generates]-> JVM node
            .. OR ..
Kotlin node  -[generates]-> JVM node
```

For each **reference** to a semantic entity in higher-level source code, if the
entity referenced is **known to be defined within the same higher-level
language**, then reference edges should target the higher-level semantic Kythe
node (e.g. Java references a Java node).  For all other references, edges should
target the corresponding JVM semantic Kythe node (e.g. Java references a JVM
node).

```
Java anchor  -[ref]-> Java semantic node
            .. OR ..
Java anchor  -[ref]-> JVM semantic node

Scala anchor -[ref]-> Scala semantic node
            .. OR ..
Scala anchor -[ref]-> JVM semantic node
```

It can be non-trivial to know whether a referenced entity was also defined
within the same language, but there are multiple possible methods to make the
determination.  First of all, it is usually trivial to determine if a node is
defined within the same compilation as a reference.  For references across
compilations, it is often possible for the build system to determine which
source file generates a `.class` file and the Kythe extractor will pack that
info into the `CompilationUnit`.  Each compiled `.class` file can also contain a
source file path (depending on the build system used).  In either of these
cases, checking the source file's path extension is usually sufficient to
determine the source language.  Finally, at the discretion of the user, it is
possible to simply default to the higher-level language of the compilation
rather than the JVM language if cross-language support is not desired.

Together, these edges tie the cross-language references of a Kythe node in one
higher-level language to the definition/references in another higher-level
language by joining across the shared JVM node.

```
Java anchor -[defines/binding]-> Java class -[generates]-> JVM class <-[ref]- Scala anchor
```

## JVM `.class` file indexer

The design of the Kythe JVM language schema is predicated by the possibility of
indexing lone `.class` files.  As such, all data used in the construction of
Kythe JVM nodes can be strictly found in compiled `.class` files as noted by the
following documentation:
https://docs.oracle.com/javase/specs/jvms/se9/html/jvms-4.html.

<!-- TODO(schroederc): additional edges from JVM classes and their enclosing `.class` and `.jar` files -->