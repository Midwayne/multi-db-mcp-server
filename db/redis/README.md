# Redis

Back to the [main README](../../README.md).

Talk to one or more Redis instances from the MCP server. Each spec connection is a named instance (`cache`, `redis-jobs`, …). Pass that name as `connection` on every tool.

## Spec

`type` is `redis`.

```yaml
connections:
  - name: cache
    type: redis
    uri: ${REDIS_URI:-redis://localhost:6379/0}
    access: read_only
```

Without `uri`, these fields build `redis://user:password@host:port/db` (port defaults to `6379`):

| Field | Meaning |
| --- | --- |
| `host`, `port`, `user`, `password` | Server and credentials |
| `db` | Logical database index (`0`, `1`, …) |

## Access

Dedicated tools (`redis_get`, `redis_set`, …) are gated by `access`. `redis_command` is always listed as a **read** tool so it stays registered for `read_only` connections; the **command name** is then classified and blocked if the mode does not allow it. Unknown commands are treated as admin.

| `access` | Allowed | Denied |
| --- | --- | --- |
| `read_only` (default) | GET, SCAN, INFO, and other read commands | SET, DEL, and admin commands |
| `read_write` | Reads plus SET/DEL and other writes | `FLUSHALL`, `CONFIG`, `ACL`, `SCRIPT`, … |
| `admin` | Writes plus dangerous commands (`FLUSHALL`, `FLUSHDB`, `CONFIG`, …) | — |

`list_permissions` with `include_server: true` reports `ACL WHOAMI` (when supported) and `DBSIZE`.

## Tools

Every tool requires `connection`. SCAN results are capped by `max_rows` (default 200).

### Read

| Tool | Arguments | What it does |
| --- | --- | --- |
| `redis_get` | `key` | `GET` a string key. Missing keys are a tool error. |
| `redis_scan` | optional `pattern` (default `*`), `count` | `SCAN` keys matching the pattern. |
| `redis_info` | optional `section` | `INFO` (optionally one section: `server`, `memory`, `keyspace`, …). |
| `redis_command` | `command`, optional `args` | Run a Redis command. Read-only connections may only run read commands. |

Shared MCP tools that also apply: `list_connections`, `list_permissions`, `ping`, `landscape` (keyspace size plus server/memory info).

### Write (`read_write` or `admin`)

| Tool | Arguments | What it does |
| --- | --- | --- |
| `redis_set` | `key`, `value` | `SET` a string key. |
| `redis_delete` | `keys` (array) | `DEL` one or more keys. |
| `redis_command` | `command`, optional `args` | Writes such as `SET` / `HSET` need `read_write`. `FLUSHALL`, `CONFIG`, `ACL`, `SCRIPT`, and unknown commands need `admin`. |

### Command classes (for `redis_command`)

Read includes `GET`, `MGET`, `EXISTS`, `SCAN`, `HGETALL`, `LRANGE`, `SMEMBERS`, `ZRANGE`, `XRANGE`, `PING`, `INFO`, `DBSIZE`, `TTL`, and similar. Write includes `SET`, `DEL`, `HSET`, `LPUSH`, `SADD`, `ZADD`, `XADD`, `EXPIRE`, and similar. Admin includes `FLUSHALL`, `FLUSHDB`, `CONFIG`, `SHUTDOWN`, `ACL`, `SCRIPT`, `EVAL`, `CLIENT`, `DEBUG`, `CLUSTER`, and anything not on the read/write lists.

## Examples

Read a key:

```json
{
  "connection": "cache",
  "key": "session:1"
}
```

Write a key (needs `read_write` or `admin`):

```json
{
  "connection": "jobs",
  "key": "worker",
  "value": "busy"
}
```

Admin flush (needs `admin`; does not affect other Redis connections):

```json
{
  "connection": "jobs",
  "command": "FLUSHALL"
}
```

Two Redis connections are separate servers (or separate logical `db` indexes) — see the mixed recipe in the [main README](../../README.md#recipes).
