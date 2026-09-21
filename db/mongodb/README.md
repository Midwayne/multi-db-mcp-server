# MongoDB

Back to the [main README](../../README.md).

Talk to one or more MongoDB deployments from the MCP server. Each spec connection is a named instance (`mongo-dev`, `app`, …). Pass that name as `connection` on every tool.

## Spec

`type` accepts `mongodb` or `mongo`. Set `database` so tools do not need `db_name` on every call.

```yaml
connections:
  - name: app
    type: mongodb
    uri: ${MONGO_URI:-mongodb://localhost:27017}
    database: app
    access: read_only
```

Without `uri`, these fields build `mongodb://user:password@host:port/database` (port defaults to `27017`):

| Field | Meaning |
| --- | --- |
| `host`, `port`, `user`, `password` | Server and credentials |
| `database` | Default database for `mongo_*` tools |

Copy-paste starter: [`examples/mongo-readwrite.yaml`](../../examples/mongo-readwrite.yaml).

## Access

| `access` | Allowed | Denied |
| --- | --- | --- |
| `read_only` (default) | Find, aggregate, count, list, indexes, schema, stats | Insert, update, delete |
| `read_write` | Reads plus insert / update / delete | — |
| `admin` | Same write tools as `read_write` (Mongo has no separate DDL tools) | — |

Write tools are only registered if **some** Mongo connection is `read_write` or `admin`. A `read_only` connection still cannot call them.

`list_permissions` with `include_server: true` runs Mongo `connectionStatus` on that instance.

## Tools

Every tool requires `connection`. `db_name` defaults to the connection’s `database`. `collection_name` is required where listed. Result rows are capped by `max_rows` (default 200).

### Read

| Tool | Arguments | What it does |
| --- | --- | --- |
| `mongo_find` | `collection_name`, optional `db_name`, `filter`, `projection`, `sort`, `limit`, `skip` | Find documents. `filter` / `projection` / `sort` are JSON objects; `filter` defaults to `{}`. |
| `mongo_aggregate` | `collection_name`, `pipeline` (JSON array), optional `db_name` | Run an aggregation pipeline. |
| `mongo_count` | `collection_name`, optional `db_name`, `filter` | Count documents matching `filter`. |
| `mongo_list_databases` | — | List database names on the server. |
| `mongo_list_collections` | optional `db_name` | List collections in a database. |
| `mongo_indexes` | `collection_name`, optional `db_name` | List indexes. |
| `mongo_schema` | `collection_name`, optional `db_name`, `sample` | Infer field types by sampling documents (default 10). |
| `mongo_stats` | optional `db_name` | `dbStats` for the database. |

Shared MCP tools that also apply: `list_connections`, `list_permissions`, `ping`, `landscape` (databases → collections).

### Write (`read_write` or `admin`)

| Tool | Arguments | What it does |
| --- | --- | --- |
| `mongo_insert` | `collection_name`, `documents` (JSON object or array), optional `db_name` | Insert one document or many. |
| `mongo_update` | `collection_name`, `filter`, `update`, optional `db_name`, `many` | Update one match, or all if `many` is true. `update` is a Mongo update document, e.g. `{"$set":{"status":"done"}}`. |
| `mongo_delete` | `collection_name`, `filter`, optional `db_name`, `many` | Delete one match, or all if `many` is true. |

## Examples

Find in the connection’s default database:

```json
{
  "connection": "app",
  "collection_name": "orders",
  "filter": "{\"status\":\"open\"}",
  "limit": 20
}
```

Insert (needs `read_write`):

```json
{
  "connection": "app",
  "collection_name": "orders",
  "documents": "{\"sku\":\"T-10\",\"status\":\"new\"}"
}
```

Two Mongo instances in one spec stay isolated by `name` — see the mixed recipe in the [main README](../../README.md#recipes).
