**Subject:** Modernizing JPMC's Developer Data Workflows with Datadatdat

Hi [Name],

Following our conversations with your technical team, I wanted to summarize how Datadatdat directly addresses the three priorities you raised:

**1. Eliminating the shared-dev-database bottleneck.** Today, 20+ developers sharing a single dev database means every outage, restore, or rehydration cascades into team-wide downtime — and developers steer clear of changes that could impact others. Datadatdat gives every developer their own isolated, fully functional database in seconds, branched from a known commit. Outages become individual, restores collapse to a single `d3 checkout`, and the cultural risk-aversion around database changes disappears.

**2. Future-proof across your evolving workspace strategy.** Datadatdat works today against container hosts *outside* the dev workspace, and the same `d3` workflow continues unchanged once containers can run *inside* workspaces. Developer experience and tooling don't change — only the runtime location does.

**3. Aligning data and application workflows.** Datadatdat brings Git semantics (clone, push, pull, commit, checkout, branch) to databases. Database state is versioned alongside code: a developer reviewing a PR can `d3 checkout` the exact data state the author used. Aligning cognitive models between data and code reduces toil, improves MTTR, and lifts developer engagement.

**How it works:**

- A small executable (`d3.exe` on Windows, `d3` on Mac/Linux) installed on the developer workspace
- A container runtime — Docker or Kubernetes — in-workspace or out
- A Datadatdat Remote Server (operationally similar to Bitbucket Data Center) that stores shared database commits

Datadatdat versions database files and container metadata, storing only deltas. Commits are fast, lightweight, and easily replicated across developers and environments.

**Database coverage includes:** PostgreSQL, MySQL, MariaDB, Microsoft SQL Server, Oracle, MongoDB, DynamoDB, CockroachDB, TiDB, Cassandra, ClickHouse, Couchbase, CouchDB, Redis, Valkey, Dragonfly, Elasticsearch, OpenSearch, Neo4j, ScyllaDB, SurrealDB, TigerGraph, Qdrant, Chroma, Weaviate, Typesense, Meilisearch, InfluxDB, QuestDB, NATS, and StarRocks.

The attached one-pager summarizes the picture. Happy to set up a working session with your platform team whenever it suits.

Best,
[Your name]
