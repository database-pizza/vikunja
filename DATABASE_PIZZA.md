# PizzaSQL backend

Vikunja has an experimental `pizzasql` database backend that targets the
[PizzaSQL](https://database.pizza) engine. It is not supported in production.

The backend bridges Vikunja's existing SQLite SQL generation to the PostgreSQL
wire protocol:

- XORM generates SQL with the SQLite dialect, mapped through a custom driver.
- Transport is the PostgreSQL simple-query protocol, through a `database/sql`
  driver in `pkg/db/pizzasql.go` built on `pgx`'s `pgconn`.
- No local SQLite file is created or read. The `database.path` option is ignored.

## Configuration

```ini
[database]
type = pizzasql
host = 127.0.0.1:15434
database = vikunja
user = vikunja
password =
sslmode = disable
```

`sslmode` defaults to `disable` for this backend. The managed database.pizza
service scopes databases as `org/db` and uses the API key as the password; a
local engine accepts any user.

## Engine requirements

This backend only works against a PizzaSQL source build that implements:

- A session-local `last_insert_rowid()` so inserts report the generated ID.
- `sqlite_master`/`sqlite_schema` catalog reads and the catalog PRAGMAs used
  during migration and schema sync.
- Recursive and non-recursive CTEs, since the permission and subscription
  queries use `WITH` and `WITH RECURSIVE`.
- `ROW_NUMBER() OVER (PARTITION BY ... ORDER BY ...)` window functions.
- `ORDER BY ... NULLS FIRST/LAST`.
- `ON CONFLICT (cols) DO UPDATE` / `DO NOTHING` against unique indexes,
  including the `excluded` pseudo-table, used for task buckets and positions.
- Enforced unique indexes and composite constraints.
- Correlated `EXISTS` subqueries in single-table `WHERE` clauses.

## Status

This backend is experimental and was validated against a local PizzaSQL engine
running Vikunja's full migration suite and task/project CRUD through the HTTP
API. It is not deployed, and the engine changes it depends on are not published.
