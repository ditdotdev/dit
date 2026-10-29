# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Per-engine client tables for the database matrix (ditdotdev/dit#262).
#
# For every image in the matrix we create one object through the engine's own
# client, run the dit flow, and read that object back at each point where the
# data must have survived (checkout, clone, migrate). A marker file in the
# volume (#261) proves the bytes moved; this proves the engine's own storage
# came through consistently and the engine answers queries afterwards.
#
# Three tables, keyed by image reference, all run through docker exec with the
# client bundled in the image so no host tooling is required:
#
#   db_ready  <container> <image>   0 when the engine accepts a query
#   db_write  <container> <image>   create the object (idempotent)
#   db_verify <container> <image>   0 when the object reads back correctly
#
# db_write and db_verify return DB_NO_CLIENT (2) for images that ship no client
# and expose no HTTP API from inside the container; those keep the marker-only
# coverage and are listed in the default case so the gap is visible. The
# object's value is always the image reference, so a verify that reads the
# wrong image's data fails too.
#
# The boot tables (db_env_args, db_dit_flags, db_docker_flags) live here too so
# the dit run in test_database, the unmanaged docker run in test_migrate and
# the local harness all start every image the same way.
#
# This file is sourced by db-matrix.bats and by the local harness
# tests/endtoend/db-matrix/check-engines.sh, which runs the three tables
# against a plain container (write, restart, verify) without the dit flow.

DB_NO_CLIENT=2

# Per-image boot requirements: environment (-e pairs, valid for both dit run
# and docker run) and flags that only one of them understands. Images whose
# entrypoint needs a command (cockroach, yugabyte, surrealdb) get it from the
# CMD in their wrapper Dockerfile in this directory, because dit run has no
# way to pass one: its trailing arguments are docker flags.
# Everything the matrix learned the hard way: without these, the engine exits
# within a second and every "is it Up" check is a race. Where the client
# tables below need credentials or an unsecured endpoint, that is set here so
# the same settings apply to every container the test starts.
db_env_args() {
  local db_version="$1"
  case "$db_version" in
    *mssql*)       echo "-e ACCEPT_EULA=Y -e MSSQL_SA_PASSWORD=YourStrong!Passw0rd -e MSSQL_PID=Developer" ;;
    *db2*)         echo "-e LICENSE=accept -e DB2INST1_PASSWORD=YourStrong!Passw0rd -e DBNAME=testdb" ;;
    *oracle*)      echo "-e ORACLE_PASSWORD=YourStrong!Passw0rd" ;;
    *minio*)       echo "-e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin" ;;
    *arangodb*)    echo "-e ARANGO_NO_AUTH=1" ;;
    postgres:*|*timescaledb*)
                   echo "-e POSTGRES_HOST_AUTH_METHOD=trust" ;;
    mysql:*)       echo "-e MYSQL_ALLOW_EMPTY_PASSWORD=yes" ;;
    mariadb:*)     echo "-e MARIADB_ALLOW_EMPTY_ROOT_PASSWORD=yes" ;;
    couchdb:*)     echo "-e COUCHDB_USER=admin -e COUCHDB_PASSWORD=password" ;;
    *influxdb*)    echo "-e INFLUXDB3_NODE_IDENTIFIER_PREFIX=node0 -e INFLUXDB3_OBJECT_STORE=file -e INFLUXDB3_DB_DIR=/var/lib/influxdb3" ;;
    *typesense*)   echo "-e TYPESENSE_DATA_DIR=/data -e TYPESENSE_API_KEY=dit-matrix" ;;
    # Single node, security off, so curl inside the container can index.
    *elasticsearch*)
                   echo "-e discovery.type=single-node -e xpack.security.enabled=false" ;;
    *opensearch*)  echo "-e discovery.type=single-node -e DISABLE_SECURITY_PLUGIN=true -e DISABLE_INSTALL_DEMO_CONFIG=true" ;;
    # cypher-shell without a forced password change.
    *neo4j*)       echo "-e NEO4J_AUTH=none" ;;
    # surreal start defaults to the in-memory store; point it at the declared
    # /data volume so there is something to version, and give it a root user
    # for the surreal sql client.
    *surrealdb*)   echo "-e SURREAL_PATH=rocksdb:/data/dit -e SURREAL_USER=root -e SURREAL_PASS=root" ;;
  esac
}

# Flags only dit run understands (port mapping is dit's own default; docker
# run publishes nothing unless asked, so the unmanaged side needs none).
db_dit_flags() {
  local db_version="$1"
  case "$db_version" in
    # Db2 needs --privileged to remount /database with suid, and port 22 /
    # SSH conflicts with the runner's sshd.
    *db2*)         echo "--privileged --disable-port-mapping" ;;
    # Oracle has no exposed ports; DynamoDB Local avoids a port clash;
    # TigerGraph runs sshd on port 22.
    *oracle*|*dynamodb*|*tigergraph*)
                   echo "--disable-port-mapping" ;;
  esac
}

# Flags only docker run understands for the unmanaged migrate source.
db_docker_flags() {
  local db_version="$1"
  case "$db_version" in
    *db2*)         echo "--privileged" ;;
  esac
}

# Seconds db_wait_ready allows an engine to accept its first query. Everything
# not listed gets DB_READY_TIMEOUT_DEFAULT. These are boot times measured on
# ubuntu-22.04 runners with a cold data directory; a restart with existing
# data is usually faster.
DB_READY_TIMEOUT_DEFAULT=180
db_ready_timeout() {
  case "$1" in
    *db2*)                       echo 900 ;;   # instance + sample db creation
    *oracle*)                    echo 600 ;;   # XE 11 and non-faststart variants
    *tigergraph*|*starrocks*)    echo 600 ;;
    *couchbase*|*scylla*|*cassandra*|*mssql*|*yugabyte*|*tidb*)
                                 echo 300 ;;
    *)                           echo "$DB_READY_TIMEOUT_DEFAULT" ;;
  esac
}

# Poll db_ready until it succeeds or the per-image timeout elapses. Each probe
# is capped so a hung client cannot stall the test.
db_wait_ready() {
  local c="$1" img="$2" timeout elapsed=0
  timeout=$(db_ready_timeout "$img")
  while ! timeout 30 bash -c 'db_ready "$@"' _ "$c" "$img" >/dev/null 2>&1; do
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "  Timeout (${timeout}s) waiting for $img in $c to accept connections"
      docker logs --tail 30 "$c" 2>&1 | sed 's/^/    | /'
      return 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
}
export -f db_wait_ready

# Readiness: one cheap query through the client. "Up" only means the process
# exists; most engines spend seconds to minutes initializing after that.
db_ready() {
  local c="$1" img="$2"
  case "$img" in
    postgres:*|*timescaledb*)
      # -h localhost skips the entrypoint's socket-only bootstrap server.
      docker exec "$c" pg_isready -h localhost -U postgres ;;
    mysql:*)
      docker exec "$c" mysql -h127.0.0.1 -uroot -e 'select 1' ;;
    mariadb:*)
      docker exec "$c" mariadb -h127.0.0.1 -uroot -e 'select 1' ;;
    *tidb*)
      docker exec "$c" mysql -h127.0.0.1 -P4000 -uroot -e 'select 1' ;;
    *starrocks*)
      docker exec "$c" mysql -h127.0.0.1 -P9030 -uroot -e 'show backends' | grep -q true ;;
    mongo:*)
      docker exec "$c" mongosh --quiet --eval 'db.runCommand({ping:1}).ok' | grep -q 1 ;;
    *mssql*)
      docker exec "$c" bash -c "\$(ls /opt/mssql-tools18/bin/sqlcmd /opt/mssql-tools/bin/sqlcmd 2>/dev/null | head -1) -S localhost -U sa -P 'YourStrong!Passw0rd' -C -Q 'select 1'" ;;
    *oracle*)
      docker exec "$c" healthcheck.sh ;;
    *db2*)
      docker exec "$c" su - db2inst1 -c 'db2 connect to testdb' ;;
    *cockroach*)
      docker exec "$c" /cockroach/cockroach sql --insecure -e 'select 1' ;;
    *yugabyte*)
      docker exec "$c" bin/ysqlsh -h "$(docker exec "$c" hostname -i)" -c 'select 1' ;;
    *cassandra*|*scylla*)
      docker exec "$c" cqlsh -e 'describe keyspaces' ;;
    *clickhouse*)
      docker exec "$c" clickhouse-client -q 'select 1' ;;
    *crate*)
      docker exec "$c" crash -c 'select 1' ;;
    *questdb*)
      docker exec "$c" curl -sfG http://localhost:9000/exec --data-urlencode 'query=select 1' ;;
    *influxdb*)
      docker exec "$c" influxdb3 show databases ;;
    *redis*|*dragonfly*)
      docker exec "$c" redis-cli ping | grep -q PONG ;;
    *valkey*)
      docker exec "$c" valkey-cli ping | grep -q PONG ;;
    *elasticsearch*|*opensearch*)
      docker exec "$c" curl -sf 'http://localhost:9200/_cluster/health?wait_for_status=yellow&timeout=5s' ;;
    *solr*)
      docker exec "$c" curl -sf 'http://localhost:8983/solr/admin/cores?action=STATUS' ;;
    couchdb:*)
      docker exec "$c" curl -sf http://admin:password@localhost:5984/_up ;;
    *neo4j*)
      docker exec "$c" cypher-shell 'return 1' ;;
    *arangodb*)
      docker exec "$c" arangosh --server.authentication false --javascript.execute-string 'db._version()' ;;
    *minio*)
      docker exec "$c" curl -sf http://localhost:9000/minio/health/live ;;
    *meilisearch*)
      docker exec "$c" curl -sf http://localhost:7700/health ;;
    *typesense*)
      docker exec "$c" curl -sf http://localhost:8108/health | grep -q '"ok":true' ;;
    *qdrant*)
      docker exec "$c" curl -sf http://localhost:6333/readyz ;;
    *chroma*)
      docker exec "$c" curl -sf http://localhost:8000/api/v2/heartbeat ;;
    *surrealdb*)
      docker exec "$c" /surreal is-ready --endpoint http://127.0.0.1:8000 ;;
    *rabbitmq*)
      docker exec "$c" rabbitmqctl await_startup ;;
    *)
      return 0 ;;
  esac
}
export -f db_ready

# Create the object. Idempotent so a retried step is harmless.
db_write() {
  local c="$1" img="$2" v="$2"
  case "$img" in
    postgres:*|*timescaledb*)
      docker exec "$c" psql -h localhost -U postgres -v ON_ERROR_STOP=1 -c "create table if not exists dit_check(v text)" -c "insert into dit_check values('$v')" ;;
    mysql:*)
      docker exec "$c" mysql -h127.0.0.1 -uroot -e "create database if not exists dit; create table if not exists dit.dit_check(v varchar(255)); insert into dit.dit_check values('$v')" ;;
    mariadb:*)
      docker exec "$c" mariadb -h127.0.0.1 -uroot -e "create database if not exists dit; create table if not exists dit.dit_check(v varchar(255)); insert into dit.dit_check values('$v')" ;;
    *tidb*)
      docker exec "$c" mysql -h127.0.0.1 -P4000 -uroot -e "create database if not exists dit; create table if not exists dit.dit_check(v varchar(255)); insert into dit.dit_check values('$v')" ;;
    *starrocks*)
      docker exec "$c" mysql -h127.0.0.1 -P9030 -uroot -e "create database if not exists dit; create table if not exists dit.dit_check(v varchar(255)) duplicate key(v) properties('replication_num'='1'); insert into dit.dit_check values('$v')" ;;
    mongo:*)
      docker exec "$c" mongosh --quiet --eval "db.getSiblingDB('dit').dit_check.replaceOne({_id:1},{_id:1,v:'$v'},{upsert:true})" ;;
    *mssql*)
      docker exec "$c" bash -c "\$(ls /opt/mssql-tools18/bin/sqlcmd /opt/mssql-tools/bin/sqlcmd 2>/dev/null | head -1) -S localhost -U sa -P 'YourStrong!Passw0rd' -C -b -Q \"if object_id('dit_check') is null create table dit_check(v nvarchar(255)); insert into dit_check values('$v')\"" ;;
    *oracle*)
      # OS-authenticated sysdba inside the container; the table lands in the
      # SYSTEM tablespace under /opt/oracle/oradata, which is the volume.
      docker exec "$c" bash -c "printf \"whenever sqlerror exit failure\ncreate table system.dit_check(v varchar2(255));\ninsert into system.dit_check values('$v');\ncommit;\n\" | sqlplus -s / as sysdba" ;;
    *db2*)
      docker exec "$c" su - db2inst1 -c "db2 connect to testdb && db2 'create table dit_check(v varchar(255))' ; db2 \"insert into dit_check values('$v')\"" ;;
    *cockroach*)
      docker exec "$c" /cockroach/cockroach sql --insecure -e "create table if not exists dit_check(v string); insert into dit_check values('$v')" ;;
    *yugabyte*)
      docker exec "$c" bin/ysqlsh -h "$(docker exec "$c" hostname -i)" -v ON_ERROR_STOP=1 -c "create table if not exists dit_check(v text)" -c "insert into dit_check values('$v')" ;;
    *cassandra*|*scylla*)
      docker exec "$c" cqlsh -e "create keyspace if not exists dit with replication={'class':'SimpleStrategy','replication_factor':1}; create table if not exists dit.dit_check(k int primary key, v text); insert into dit.dit_check(k,v) values(1,'$v')" \
        && docker exec "$c" nodetool flush ;;
    *clickhouse*)
      docker exec "$c" clickhouse-client -q "create table if not exists dit_check(v String) engine=MergeTree order by v" \
        && docker exec "$c" clickhouse-client -q "insert into dit_check values('$v')" ;;
    *crate*)
      docker exec "$c" crash -c "create table if not exists dit_check(v text)" \
        && docker exec "$c" crash -c "insert into dit_check values('$v')" \
        && docker exec "$c" crash -c "refresh table dit_check" ;;
    *questdb*)
      docker exec "$c" curl -sfG http://localhost:9000/exec --data-urlencode "query=create table if not exists dit_check(v string)" \
        && docker exec "$c" curl -sfG http://localhost:9000/exec --data-urlencode "query=insert into dit_check values('$v')" ;;
    *influxdb*)
      docker exec "$c" influxdb3 create database dit >/dev/null 2>&1
      docker exec "$c" influxdb3 write --database dit "dit_check,k=1 v=\"$v\"" ;;
    *redis*|*dragonfly*)
      docker exec "$c" redis-cli set dit_check "$v" && docker exec "$c" redis-cli save ;;
    *valkey*)
      docker exec "$c" valkey-cli set dit_check "$v" && docker exec "$c" valkey-cli save ;;
    *elasticsearch*|*opensearch*)
      docker exec "$c" curl -sf -XPUT 'http://localhost:9200/dit_check/_doc/1?refresh=true' -H 'Content-Type: application/json' -d "{\"v\":\"$v\"}" ;;
    *solr*)
      # Core admin API rather than the solr CLI: the CLI's flags changed
      # between Solr 9 and 10, the API did not.
      docker exec "$c" curl -sf 'http://localhost:8983/solr/admin/cores?action=CREATE&name=dit_check&configSet=_default' >/dev/null 2>&1
      docker exec "$c" curl -sf 'http://localhost:8983/solr/dit_check/update?commit=true' -H 'Content-Type: application/json' -d "[{\"id\":\"1\",\"v_s\":\"$v\"}]" ;;
    couchdb:*)
      docker exec "$c" curl -sf -XPUT http://admin:password@localhost:5984/dit_check >/dev/null 2>&1
      docker exec "$c" curl -sf -XPUT http://admin:password@localhost:5984/dit_check/1 -d "{\"v\":\"$v\"}" ;;
    *neo4j*)
      docker exec "$c" cypher-shell "merge (n:DitCheck {k:1}) set n.v='$v'" ;;
    *arangodb*)
      docker exec "$c" arangosh --server.authentication false --javascript.execute-string "if(!db._collection('dit_check')) db._create('dit_check'); db.dit_check.save({_key:'1', v:'$v'}, {overwrite:true});" ;;
    *minio*)
      docker exec "$c" mc alias set dit http://localhost:9000 minioadmin minioadmin >/dev/null \
        && docker exec "$c" mc mb --ignore-existing dit/dit-check >/dev/null \
        && docker exec "$c" sh -c "printf '%s' '$v' | mc pipe dit/dit-check/check" ;;
    *meilisearch*)
      # Document indexing is asynchronous; wait for the task so the write is
      # on disk before the container is stopped.
      docker exec "$c" curl -sf -XPOST http://localhost:7700/indexes/dit_check/documents -H 'Content-Type: application/json' -d "[{\"id\":1,\"v\":\"$v\"}]" >/dev/null \
        && docker exec "$c" sh -c 'for i in $(seq 1 60); do curl -sf http://localhost:7700/indexes/dit_check/documents/1 >/dev/null && exit 0; sleep 1; done; exit 1' ;;
    *typesense*)
      docker exec "$c" curl -sf -XPOST http://localhost:8108/collections -H 'X-TYPESENSE-API-KEY: dit-matrix' -H 'Content-Type: application/json' -d '{"name":"dit_check","fields":[{"name":"v","type":"string"}]}' >/dev/null 2>&1
      docker exec "$c" curl -sf -XPOST 'http://localhost:8108/collections/dit_check/documents?action=upsert' -H 'X-TYPESENSE-API-KEY: dit-matrix' -H 'Content-Type: application/json' -d "{\"id\":\"1\",\"v\":\"$v\"}" ;;
    *qdrant*)
      docker exec "$c" curl -sf -XPUT http://localhost:6333/collections/dit_check -H 'Content-Type: application/json' -d '{"vectors":{"size":2,"distance":"Dot"}}' >/dev/null 2>&1
      docker exec "$c" curl -sf -XPUT 'http://localhost:6333/collections/dit_check/points?wait=true' -H 'Content-Type: application/json' -d "{\"points\":[{\"id\":1,\"vector\":[0.1,0.2],\"payload\":{\"v\":\"$v\"}}]}" ;;
    *chroma*)
      docker exec "$c" sh -c "id=\$(curl -sf -XPOST http://localhost:8000/api/v2/tenants/default_tenant/databases/default_database/collections -H 'Content-Type: application/json' -d '{\"name\":\"dit_check\",\"get_or_create\":true}' | sed -n 's/.*\"id\":\"\([^\"]*\)\".*/\1/p') && [ -n \"\$id\" ] && curl -sf -XPOST \"http://localhost:8000/api/v2/tenants/default_tenant/databases/default_database/collections/\$id/upsert\" -H 'Content-Type: application/json' -d '{\"ids\":[\"1\"],\"embeddings\":[[0.1,0.2]],\"metadatas\":[{\"v\":\"$v\"}]}'" ;;
    *surrealdb*)
      printf "define namespace if not exists dit; use ns dit; define database if not exists dit; use db dit; upsert dit_check:1 set v = '%s';\n" "$v" \
        | docker exec -i "$c" /surreal sql --endpoint http://127.0.0.1:8000 -u root -p root --ns dit --db dit --hide-welcome ;;
    *rabbitmq*)
      # The image has no AMQP client (rabbitmqadmin needs the management
      # plugin), so the object is a vhost: a durable entry in the broker's
      # metadata store under /var/lib/rabbitmq. The vhost name carries the
      # image reference, slashes replaced since vhost names cannot nest them.
      docker exec "$c" rabbitmqctl -q add_vhost "dit_check_${v//\//_}" >/dev/null 2>&1 || true
      docker exec "$c" rabbitmqctl -q list_vhosts | grep -qF "dit_check_${v//\//_}" ;;
    # No client in the image and no HTTP API reachable without one:
    #   dynamodb-local  Java image, no shell beyond the JVM, SDK-only wire protocol
    #   nats            scratch image: nats-server only, no nats CLI
    #   weaviate        distroless: no shell, no curl
    #   kafka           no VOLUME declared, so there is nothing to carry anyway
    #   couchbase       needs cluster init through the REST API before a bucket
    #                   can exist; see #262 follow-up
    #   tigergraph      gsql requires a schema, loading job and graph; see #262
    *)
      return "$DB_NO_CLIENT" ;;
  esac
}
export -f db_write

# Read the object back; non-zero if it is missing or does not carry the image
# reference it was written with.
db_verify() {
  local c="$1" img="$2" v="$2"
  case "$img" in
    postgres:*|*timescaledb*)
      docker exec "$c" psql -h localhost -U postgres -tAc "select v from dit_check" | grep -qxF "$v" ;;
    mysql:*)
      docker exec "$c" mysql -h127.0.0.1 -uroot -N -e "select v from dit.dit_check" | grep -qxF "$v" ;;
    mariadb:*)
      docker exec "$c" mariadb -h127.0.0.1 -uroot -N -e "select v from dit.dit_check" | grep -qxF "$v" ;;
    *tidb*)
      docker exec "$c" mysql -h127.0.0.1 -P4000 -uroot -N -e "select v from dit.dit_check" | grep -qxF "$v" ;;
    *starrocks*)
      docker exec "$c" mysql -h127.0.0.1 -P9030 -uroot -N -e "select v from dit.dit_check" | grep -qxF "$v" ;;
    mongo:*)
      docker exec "$c" mongosh --quiet --eval "db.getSiblingDB('dit').dit_check.findOne({_id:1}).v" | grep -qxF "$v" ;;
    *mssql*)
      docker exec "$c" bash -c "\$(ls /opt/mssql-tools18/bin/sqlcmd /opt/mssql-tools/bin/sqlcmd 2>/dev/null | head -1) -S localhost -U sa -P 'YourStrong!Passw0rd' -C -h -1 -W -Q 'set nocount on; select v from dit_check'" | grep -qxF "$v" ;;
    *oracle*)
      docker exec "$c" bash -c "printf 'set heading off feedback off pagesize 0\nselect v from system.dit_check;\n' | sqlplus -s / as sysdba" | grep -qxF "$v" ;;
    *db2*)
      docker exec "$c" su - db2inst1 -c "db2 connect to testdb >/dev/null && db2 -x 'select v from dit_check'" | grep -qF "$v" ;;
    *cockroach*)
      docker exec "$c" /cockroach/cockroach sql --insecure --format=tsv -e "select v from dit_check" | grep -qxF "$v" ;;
    *yugabyte*)
      docker exec "$c" bin/ysqlsh -h "$(docker exec "$c" hostname -i)" -tAc "select v from dit_check" | grep -qxF "$v" ;;
    *cassandra*|*scylla*)
      docker exec "$c" cqlsh -e "select v from dit.dit_check where k=1" | grep -qF "$v" ;;
    *clickhouse*)
      docker exec "$c" clickhouse-client -q "select v from dit_check" | grep -qxF "$v" ;;
    *crate*)
      docker exec "$c" crash -c "select v from dit_check" | grep -qF "$v" ;;
    *questdb*)
      docker exec "$c" curl -sfG http://localhost:9000/exec --data-urlencode "query=select v from dit_check" | grep -qF "\"$v\"" ;;
    *influxdb*)
      docker exec "$c" influxdb3 query --database dit "select v from dit_check" | grep -qF "$v" ;;
    *redis*|*dragonfly*)
      docker exec "$c" redis-cli get dit_check | grep -qxF "$v" ;;
    *valkey*)
      docker exec "$c" valkey-cli get dit_check | grep -qxF "$v" ;;
    *elasticsearch*|*opensearch*)
      docker exec "$c" curl -sf http://localhost:9200/dit_check/_doc/1 | grep -qF "\"v\":\"$v\"" ;;
    *solr*)
      docker exec "$c" curl -sf 'http://localhost:8983/solr/dit_check/select?q=id:1' | grep -qF "\"v_s\":\"$v\"" ;;
    couchdb:*)
      docker exec "$c" curl -sf http://admin:password@localhost:5984/dit_check/1 | grep -qF "\"v\":\"$v\"" ;;
    *neo4j*)
      docker exec "$c" cypher-shell --format plain "match (n:DitCheck {k:1}) return n.v" | grep -qF "\"$v\"" ;;
    *arangodb*)
      docker exec "$c" arangosh --server.authentication false --javascript.execute-string "print(db.dit_check.document('1').v)" | grep -qxF "$v" ;;
    *minio*)
      docker exec "$c" mc alias set dit http://localhost:9000 minioadmin minioadmin >/dev/null \
        && docker exec "$c" mc cat dit/dit-check/check | grep -qxF "$v" ;;
    *meilisearch*)
      docker exec "$c" curl -sf http://localhost:7700/indexes/dit_check/documents/1 | grep -qF "\"v\":\"$v\"" ;;
    *typesense*)
      docker exec "$c" curl -sf http://localhost:8108/collections/dit_check/documents/1 -H 'X-TYPESENSE-API-KEY: dit-matrix' | grep -qF "\"v\":\"$v\"" ;;
    *qdrant*)
      docker exec "$c" curl -sf http://localhost:6333/collections/dit_check/points/1 | grep -qF "\"v\":\"$v\"" ;;
    *chroma*)
      docker exec "$c" sh -c "id=\$(curl -sf http://localhost:8000/api/v2/tenants/default_tenant/databases/default_database/collections/dit_check | sed -n 's/.*\"id\":\"\([^\"]*\)\".*/\1/p') && [ -n \"\$id\" ] && curl -sf -XPOST \"http://localhost:8000/api/v2/tenants/default_tenant/databases/default_database/collections/\$id/get\" -H 'Content-Type: application/json' -d '{\"ids\":[\"1\"],\"include\":[\"metadatas\"]}'" | grep -qF "\"v\":\"$v\"" ;;
    *surrealdb*)
      printf "use ns dit; use db dit; select v from dit_check:1;\n" \
        | docker exec -i "$c" /surreal sql --endpoint http://127.0.0.1:8000 -u root -p root --ns dit --db dit --hide-welcome | grep -qF "$v" ;;
    *rabbitmq*)
      docker exec "$c" rabbitmqctl -q list_vhosts | grep -qxF "dit_check_${v//\//_}" ;;
    *)
      return "$DB_NO_CLIENT" ;;
  esac
}
export -f db_verify
