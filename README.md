# Multi-Database MCP Server

Talk to **MongoDB**, **PostgreSQL**, and **Redis** from one MCP server. You describe your databases in a YAML file; the agent lists them, checks permissions, and queries each one by name.

**Per-database tools and access:** [MongoDB](db/mongodb/README.md) · [PostgreSQL](db/postgres/README.md) · [Redis](db/redis/README.md)

## 5-minute setup

**1. Build the binary**

```bash
git clone https://github.com/Midwayne/multi-db-mcp-server.git
cd multi-db-mcp-server
go build -o dbmcp .
```

**2. Create `spec.yaml` next to the binary** (or anywhere you like)

Start with one database. Copy a block from [Recipes](#recipes) or use the annotated [`spec.example.yaml`](spec.example.yaml).

Minimal Postgres example:

```yaml
connections:
  - name: app
    type: postgres
    uri: postgres://USER:PASSWORD@localhost:5432/app?sslmode=disable
    access: read_only
```

**3. Point your MCP client at it**

Cursor (`~/.cursor/mcp.json` or `.cursor/mcp.json` in a project):

```json
{
  "mcpServers": {
    "databases": {
      "command": "/ABS/PATH/TO/dbmcp",
      "args": ["-spec", "/ABS/PATH/TO/spec.yaml"]
    }
  }
}
```

Use **absolute paths**. Restart Cursor (or reload MCP servers), then ask:

> List my databases and their permissions.

The agent should call `list_connections` and `list_permissions`.

---

## Recipes

Copy one of these into `spec.yaml`. Change the `name`, `uri`, and `access`. Add as many `connections` entries as you need — mixed engines in one file is the normal case.

### Postgres only (read-only)

Full tool list: [PostgreSQL](db/postgres/README.md).

```yaml
connections:
  - name: app
    type: postgres
    uri: postgres://USER:PASSWORD@HOST:5432/DBNAME?sslmode=disable
    access: read_only
```

### MongoDB only (read-write)

Full tool list: [MongoDB](db/mongodb/README.md).

```yaml
connections:
  - name: app
    type: mongodb
    uri: mongodb://USER:PASSWORD@HOST:27017
    database: app
    access: read_write
```

### Redis only (read-only)

Full tool list: [Redis](db/redis/README.md).

```yaml
connections:
  - name: cache
    type: redis
    uri: redis://HOST:6379/0
    access: read_only
```

### Mixed: analytics (read-only) + app (read-write) + cache

```yaml
defaults:
  access: read_only
  max_rows: 200

connections:
  - name: analytics
    type: postgres
    uri: postgres://readonly@warehouse:5432/analytics?sslmode=require
    access: read_only

  - name: app
    type: mongodb
    uri: mongodb://app:SECRET@mongo:27017
    database: app
    access: read_write

  - name: cache
    type: redis
    uri: redis://:SECRET@redis:6379/0
    access: read_only
```

Ready-made files you can copy:

- [`spec.example.yaml`](spec.example.yaml) — annotated file with all three engines
- [`examples/postgres-readonly.yaml`](examples/postgres-readonly.yaml) — see [PostgreSQL tools](db/postgres/README.md)
- [`examples/mongo-readwrite.yaml`](examples/mongo-readwrite.yaml) — see [MongoDB tools](db/mongodb/README.md)
- [`examples/mixed.yaml`](examples/mixed.yaml)

---

## Keep passwords out of the spec

Use `${ENV_VAR}` in the spec. They are expanded when the server starts.

```yaml
connections:
  - name: app
    type: postgres
    uri: ${POSTGRES_URI}
    access: read_only
```

```json
{
  "mcpServers": {
    "databases": {
      "command": "/ABS/PATH/TO/dbmcp",
      "args": ["-spec", "/ABS/PATH/TO/spec.yaml"],
      "env": {
        "POSTGRES_URI": "postgres://USER:PASSWORD@localhost:5432/app?sslmode=disable"
      }
    }
  }
}
```

Defaults if an env var is unset: `${POSTGRES_URI:-postgres://localhost:5432/app}`.

---

## Spec reference

Only `connections` is required. Everything else has defaults.

### Connection (each item under `connections`)

| Field | Required | What it is |
| --- | --- | --- |
| `name` | yes | Short label the agent passes as `connection`. Use `app`, `analytics`, `prod-pg`. |
| `type` | yes | `mongodb` (or `mongo`), `postgres` (or `pg`), `redis` — see [MongoDB](db/mongodb/README.md), [PostgreSQL](db/postgres/README.md), [Redis](db/redis/README.md) |
| `uri` | yes, unless you use host fields | Connection string |
| `access` | no | `read_only` (default), `read_write`, or `admin` |
| `database` | no | Default Mongo/Postgres database for tools |
| `enabled` | no | `false` to skip this connection without deleting it |
| `max_rows` | no | Cap on query results (default `200`) |
| `connect_timeout` | no | e.g. `10s` |
| `fail_on_connect_error` | no | `false` if this DB might be down at startup |

If `uri` is omitted, build it from discrete fields:

| Field | Used by |
| --- | --- |
| `host`, `port`, `user`, `password` | all |
| `database`, `ssl_mode` | Postgres (and Mongo `database`) |
| `db` | Redis logical database index (0, 1, …) |

```yaml
- name: app
  type: postgres
  host: localhost
  port: 5432
  user: postgres
  password: ${PGPASSWORD}
  database: app
  ssl_mode: disable
  access: read_only
```

### Access modes

| `access` | The agent can | The agent cannot |
| --- | --- | --- |
| `read_only` | List, describe, SELECT / find / GET | INSERT, UPDATE, DELETE, SET, DDL |
| `read_write` | Read plus data changes | DROP/CREATE (Postgres), FLUSHALL (Redis) |
| `admin` | Everything above plus DDL and dangerous Redis commands | — |

Postgres `read_only` also sets `default_transaction_read_only` on the session.

Prefer a read-only database user **and** `access: read_only` for production analytics.

### Server and defaults

```yaml
server:
  serve_mode: stdio    # stdio (Cursor/Claude), or http / sse
  port: 8080           # only used for http and sse

defaults:
  access: read_only    # used when a connection omits access
  max_rows: 200
  connect_timeout: 10s
  connect_on_start: true
  fail_on_connect_error: true
```

### Limit tools (optional)

By default every tool allowed by `type` + `access` is enabled.

```yaml
# Hide a tool everywhere
tools:
  exclude: [mongo_aggregate, redis_command]

# Or allow only these tools globally
tools:
  include: [list_connections, list_permissions, ping, postgres_query]
```

Per connection:

```yaml
- name: prod
  type: postgres
  uri: ${POSTGRES_URI}
  access: read_only
  tools:
    include: [postgres_query, postgres_list_tables, postgres_describe_table]
```

---

## How the agent uses your spec

1. `list_connections` — every database: name, engine, `can_read` / `can_write` / `can_admin`
2. `list_permissions` — omit `connection` for all, or pass one name. Add `include_server: true` for live roles/ACL
3. Engine tools with that same `connection` name, for example `postgres_query` / `mongo_find` / `redis_get`

Shared tools on every spec: `list_connections`, `list_permissions`, `ping`, `landscape`. Each engine’s tools, arguments, and access rules:

| Engine | Guide |
| --- | --- |
| MongoDB | [db/mongodb/README.md](db/mongodb/README.md) — find, aggregate, insert/update/delete |
| PostgreSQL | [db/postgres/README.md](db/postgres/README.md) — query, execute, list/describe tables |
| Redis | [db/redis/README.md](db/redis/README.md) — GET/SET, SCAN, INFO, `redis_command` |

---

## Run without an MCP client

```bash
./dbmcp -spec /path/to/spec.yaml
```

If you omit `-spec`, the server looks for `DBMCP_SPEC`, then `./spec.yaml`, `./spec.yml`, or `./spec.json`.

| `serve_mode` | When to use it |
| --- | --- |
| `stdio` | Cursor, Claude Desktop, and most MCP clients (default) |
| `http` | Streamable HTTP on `port` |
| `sse` | Server-sent events on `port` |

---

## Troubleshooting

**`no spec file found`**  
Pass `-spec /absolute/path/spec.yaml` in the MCP `args`, or put `spec.yaml` in the process working directory.

**`Error initializing connections` / connection refused**  
The URI host/port is wrong, or the database is not running. For a laptop spec where some DBs are optional:

```yaml
defaults:
  fail_on_connect_error: false
```

**`${POSTGRES_URI}` is empty**  
The MCP client did not pass that env var. Put it under `env` in `mcp.json`, not only in your shell.

**Agent cannot write**  
The connection `access` is `read_only`, or the tool is in `exclude`. Check with `list_permissions` and that `connection` name.

**Wrong database**  
Every tool takes `connection` matching the spec `name`. Run `list_connections` if unsure.

**Claude Desktop** uses the same JSON shape in `claude_desktop_config.json` (`command` + `args` + optional `env`).

---

## For contributors

```
spec.yaml  →  spec.Load  →  db.Open (engine registry)  →  db.Registry  →  MCP tools
```

Each database kind is a self-contained plugin:

1. `db/<kind>/` — implement `db.Adapter`. In `init`, call `spec.RegisterKind` (type aliases + host/port URI builder) and `db.Register` (opener).
2. `connect/connect.go` — blank-import that package. This is the only composition-root edit.
3. `server/tools_<kind>.go` — MCP tools. In `init`, call `RegisterEngineTools`. Add or drop a feature by adding or removing one `toolSpec` in that file. Document the kind in `db/<kind>/README.md` and link it from this file.

Removing a kind is the reverse: delete the adapter package, delete the tools file, remove the blank import, remove the guide link.

Access modes, `list_connections`, and `list_permissions` stay generic. Optional live identity is `db.IdentityProvider`; permission footnotes hang off the tool spec (`Note`), not a type switch.

```bash
go test ./...
go test -short ./...   # skip live integration engines
```

