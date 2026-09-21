# PostgreSQL

Back to the [main README](../../README.md).

Talk to one or more Postgres databases from the MCP server. Each spec connection is a named instance (`analytics`, `pg-app`, …). Pass that name as `connection` on every tool.

## Spec

`type` accepts `postgres`, `postgresql`, or `pg`.

```yaml
connections:
  - name: app
    type: postgres
    uri: ${POSTGRES_URI:-postgres://USER:PASSWORD@localhost:5432/app?sslmode=disable}
    access: read_only
```

Without `uri`, these fields build `postgres://user:password@host:port/database?sslmode=…` (port defaults to `5432`):

| Field | Meaning |
| --- | --- |
| `host`, `port`, `user`, `password` | Server and credentials |
| `database` | Database name in the URI path |
| `ssl_mode` | e.g. `disable`, `require` |

Copy-paste starter: [`examples/postgres-readonly.yaml`](../../examples/postgres-readonly.yaml).

## Access

SQL is classified by leading keyword (comments stripped; multiple statements rejected). Unknown keywords are treated as admin.

| `access` | Allowed | Denied |
| --- | --- | --- |
| `read_only` (default) | `SELECT` / `WITH`…`SELECT`, `SHOW`, `EXPLAIN`, list/describe tools | `INSERT`/`UPDATE`/`DELETE` and DDL |
| `read_write` | Reads plus DML via `postgres_execute` | `CREATE`/`DROP`/`ALTER`/`TRUNCATE` and other DDL |
| `admin` | DML and DDL | — |

`postgres_query` only runs read statements. Writes go through `postgres_execute`.

`read_only` also sets `default_transaction_read_only` on the session, so the server rejects writes even if classification missed one.

Prefer a read-only database user **and** `access: read_only` for production analytics.

`list_permissions` with `include_server: true` reports the live role (`current_user`, superuser, `CREATE` privilege, and similar).

## Tools

Every tool requires `connection`. Query rows are capped by `max_rows` (default 200). Bind parameters are a JSON array string, e.g. `"[1, \"active\"]"`.

### Read

| Tool | Arguments | What it does |
| --- | --- | --- |
| `postgres_query` | `sql`, optional `params` | Run a read-only statement (`SELECT`, `WITH`…`SELECT`, `EXPLAIN`, `SHOW`, …). |
| `postgres_list_databases` | — | List non-template databases. |
| `postgres_list_schemas` | — | List schemas. |
| `postgres_list_tables` | optional `schema` | List tables (excludes `pg_catalog` / `information_schema`). |
| `postgres_describe_table` | `table`, optional `schema` (default `public`) | Column names, types, nullability, defaults. |
| `postgres_stats` | — | Current database, user, version, and size. |

Shared MCP tools that also apply: `list_connections`, `list_permissions`, `ping`, `landscape` (schema → tables).

### Write

| Tool | `access` | Arguments | What it does |
| --- | --- | --- | --- |
| `postgres_execute` | `read_write` | `sql`, optional `params` | `INSERT` / `UPDATE` / `DELETE` (and similar DML). |
| `postgres_execute` | `admin` | `sql`, optional `params` | DML plus DDL (`CREATE`, `DROP`, `ALTER`, …). |

## Examples

Parameterized select:

```json
{
  "connection": "app",
  "sql": "SELECT id, name FROM items WHERE id = $1",
  "params": "[1]"
}
```

Insert (needs `read_write`):

```json
{
  "connection": "app",
  "sql": "INSERT INTO items (id, name) VALUES ($1, $2)",
  "params": "[2, \"gadget\"]"
}
```

Create a table (needs `admin`):

```json
{
  "connection": "pg-admin",
  "sql": "CREATE TABLE extra (id INT PRIMARY KEY, note TEXT)"
}
```

Two Postgres connections can point at different databases (or the same database with different `access`) — see the mixed recipe in the [main README](../../README.md#recipes).
