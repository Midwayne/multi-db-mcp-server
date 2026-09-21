# Multi-Database MCP Server

A single MCP server that talks to **MongoDB**, **PostgreSQL**, and **Redis** from one spec file. Each connection has its own URI and access mode (`read_only`, `read_write`, or `admin`). Tools are registered from the spec: if every Mongo connection is read-only, write tools such as `mongo_insert` are not exposed. Write tools still check the target connection at call time, so mixed-mode setups stay safe.

## Architecture

```
spec.yaml  -->  spec.Load / Normalize
                    |
                    v
              connect.Open  -->  db.Adapter
                    |              |- mongodb
                    |              |- postgres
                    |              `- redis
                    v
              db.Registry  -->  MCP tools + resources
```

- **`spec`**: YAML/JSON document, env expansion, defaults, validation
- **`access`**: connection modes plus SQL and Redis command classification
- **`db.Adapter`**: common contract (`Ping`, `Landscape`, access, tool allowlists)
- **engine packages**: Mongo/Postgres/Redis implement the adapter plus engine-specific operations
- **`server`**: MCP tool catalog, filtered by spec and access mode

Adding another engine means implementing `db.Adapter`, handling it in `connect.Open`, and adding tools under `server/`.

## Spec file

Copy `spec.example.yaml` to `spec.yaml`. Secrets can stay in the environment and be referenced as `${MONGO_URI}` or `${PGPASSWORD:-}`.

Access modes:

| Mode | Allowed |
| --- | --- |
| `read_only` | Inspect and query. Postgres also sets `default_transaction_read_only`. |
| `read_write` | Read plus data changes (insert/update/delete, Redis SET/DEL). |
| `admin` | Read/write plus DDL and dangerous Redis commands (`FLUSHALL`, `CONFIG`, ...). |

Global `tools.include` / `tools.exclude` and per-connection lists further restrict what is registered and what a connection may run.

## Run

```bash
go build -o dbmcp .
./dbmcp -spec spec.yaml
```

`--spec` can be omitted if `DBMCP_SPEC` is set or `spec.yaml` / `spec.yml` / `spec.json` exists in the working directory.

Serve modes: `stdio` (default), `http`, and `sse`.

## MCP client

```json
{
  "mcpServers": {
    "databases": {
      "command": "/path/to/dbmcp",
      "args": ["-spec", "/path/to/spec.yaml"]
    }
  }
}
```

Call `list_connections` to see every spec database, or `list_permissions` to view access modes and allowed tools. Omit `connection` to inspect all databases, or pass `connection` for one. Then call engine tools with that same name (`mongo_find`, `postgres_query`, `redis_get`, and so on). Set `include_server: true` on `list_permissions` to also fetch live engine identity (Mongo roles, Postgres role flags, Redis ACL user).
