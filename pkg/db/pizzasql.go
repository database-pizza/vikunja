// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"xorm.io/xorm"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/schemas"

	"code.vikunja.io/api/pkg/config"
)

// pizzasqlDriverName is the database/sql and xorm driver name for the
// database.pizza transport. It speaks SQLite-dialect SQL over the PostgreSQL
// simple-query protocol and never opens a local SQLite file.
const pizzasqlDriverName = "pizzasql"

const (
	pizzasqlCloseTimeout  = 5 * time.Second
	pizzasqlFinishTimeout = 30 * time.Second
)

const (
	oidDate        = 1082
	oidTimestamp   = 1114
	oidTimestamptz = 1184
	oidTime        = 1266
)

func init() {
	sql.Register(pizzasqlDriverName, &pizzaDriver{})
	// Map the pizzasql transport onto xorm's SQLite dialect so Vikunja emits
	// SQLite-style SQL while the transport stays PostgreSQL.
	dialects.RegisterDriver(pizzasqlDriverName, &pizzaXormDriver{Driver: dialects.QueryDriver("sqlite3")})
}

// pizzaXormDriver delegates every driver feature to the built-in sqlite3
// driver and only overrides URI parsing so xorm selects the SQLite dialect.
type pizzaXormDriver struct {
	dialects.Driver
}

func (d *pizzaXormDriver) Parse(driverName, dataSourceName string) (*dialects.URI, error) {
	return &dialects.URI{DBType: schemas.SQLITE, DBName: dataSourceName}, nil
}

func initPizzasqlEngine() (*xorm.Engine, error) {
	connStr, err := pizzasqlConnString()
	if err != nil {
		return nil, err
	}

	engine, err := xorm.NewEngine(pizzasqlDriverName, connStr)
	if err != nil {
		return nil, err
	}
	engine.SetMaxOpenConns(config.DatabaseMaxOpenConnections.GetInt())
	engine.SetMaxIdleConns(config.DatabaseMaxIdleConnections.GetInt())
	maxLifetime, err := time.ParseDuration(strconv.Itoa(config.DatabaseMaxConnectionLifetime.GetInt()) + `ms`)
	if err != nil {
		return nil, err
	}
	engine.SetConnMaxLifetime(maxLifetime)
	return engine, nil
}

// pizzasqlConnString builds a PostgreSQL URL that the pizzasql transport parses
// with pgconn. The managed service scopes databases as "org/db" and uses the
// API key as the password; a local engine accepts any user.
func pizzasqlConnString() (string, error) {
	host, port := parsePostgreSQLHostPort(config.DatabaseHost.GetString())
	sslmode := config.DatabaseSslMode.GetString()
	if sslmode == "" {
		sslmode = "disable"
	}

	u := &url.URL{
		Scheme:   "postgres",
		Host:     host + ":" + port,
		Path:     "/" + config.DatabaseDatabase.GetString(),
		RawQuery: url.Values{"sslmode": {sslmode}}.Encode(),
	}
	if user := config.DatabaseUser.GetString(); user != "" || config.DatabasePassword.GetString() != "" {
		u.User = url.UserPassword(user, config.DatabasePassword.GetString())
	}
	return u.String(), nil
}

var (
	_ driver.Driver           = (*pizzaDriver)(nil)
	_ driver.DriverContext    = (*pizzaDriver)(nil)
	_ driver.Connector        = (*connector)(nil)
	_ driver.Conn             = (*connection)(nil)
	_ driver.ConnBeginTx      = (*connection)(nil)
	_ driver.ExecerContext    = (*connection)(nil)
	_ driver.QueryerContext   = (*connection)(nil)
	_ driver.Pinger           = (*connection)(nil)
	_ driver.SessionResetter  = (*connection)(nil)
	_ driver.Validator        = (*connection)(nil)
	_ driver.Stmt             = (*statement)(nil)
	_ driver.StmtExecContext  = (*statement)(nil)
	_ driver.StmtQueryContext = (*statement)(nil)
	_ driver.Tx               = (*transaction)(nil)
	_ driver.Result           = result{}
	_ driver.Rows             = (*rows)(nil)
)

type pizzaDriver struct{}

type connector struct {
	config *pgconn.Config
}

type connection struct {
	pg *pgconn.PgConn
}

type statement struct {
	c     *connection
	query string
}

type transaction struct {
	c *connection
}

type result struct {
	id, affected int64
}

type rows struct {
	fields []pgconn.FieldDescription
	values [][][]byte
	pos    int
}

func (d *pizzaDriver) Open(dsn string) (driver.Conn, error) {
	c, err := d.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}
	return c.Connect(context.Background())
}

func (*pizzaDriver) OpenConnector(dsn string) (driver.Connector, error) {
	cfg, err := pgconn.ParseConfig(dsn)
	if err != nil {
		return nil, errors.New("pizzasql: invalid connection configuration")
	}
	return &connector{config: cfg}, nil
}

func (*connector) Driver() driver.Driver { return &pizzaDriver{} }

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	pg, err := pgconn.ConnectConfig(ctx, c.config.Copy())
	if err != nil {
		return nil, sanitizeConnectError(err)
	}
	conn := &connection{pg: pg}
	if err := conn.checkCapability(ctx); err != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), pizzasqlCloseTimeout)
		defer cancel()
		_ = pg.Close(closeCtx)
		return nil, err
	}
	return conn, nil
}

func sanitizeConnectError(err error) error {
	var connErr *pgconn.ConnectError
	if !errors.As(err, &connErr) {
		return errors.New("pizzasql: connection failed")
	}
	var pgErr *pgconn.PgError
	if errors.As(connErr, &pgErr) {
		return fmt.Errorf("pizzasql: connection failed (SQLSTATE %s)", pgErr.Code)
	}
	return errors.New("pizzasql: connection failed; check address, TLS settings, and credentials")
}

func (c *connection) checkCapability(ctx context.Context) error {
	r, err := c.query(ctx, "SELECT last_insert_rowid()", nil)
	if err != nil {
		return err
	}
	if len(r.Rows) != 1 || len(r.Rows[0]) != 1 || r.Rows[0][0] == nil {
		return errors.New("pizzasql: engine does not support a session-local last_insert_rowid")
	}
	if _, err := strconv.ParseInt(string(r.Rows[0][0]), 10, 64); err != nil {
		return fmt.Errorf("pizzasql: engine returned a non-integer last_insert_rowid: %w", err)
	}
	return nil
}

func (c *connection) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), pizzasqlCloseTimeout)
	defer cancel()
	return c.pg.Close(ctx)
}

func (c *connection) IsValid() bool {
	return !c.pg.IsClosed() && c.pg.TxStatus() == 'I'
}

func (c *connection) ResetSession(context.Context) error {
	if !c.IsValid() {
		return driver.ErrBadConn
	}
	return nil
}

func (c *connection) Ping(ctx context.Context) error {
	return c.checkCapability(ctx)
}

func (c *connection) Prepare(query string) (driver.Stmt, error) {
	return &statement{c: c, query: query}, nil
}

func (s *statement) Close() error  { return nil }
func (s *statement) NumInput() int { return -1 }

func named(args []driver.Value) []driver.NamedValue {
	values := make([]driver.NamedValue, len(args))
	for i, v := range args {
		values[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	return values
}

func (s *statement) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), named(args))
}

func (s *statement) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), named(args))
}

func (s *statement) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.c.ExecContext(ctx, s.query, args)
}

func (s *statement) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.c.QueryContext(ctx, s.query, args)
}

func (c *connection) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *connection) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if opts.Isolation != 0 || opts.ReadOnly {
		return nil, errors.New("pizzasql: explicit transaction isolation and read-only transactions are unsupported")
	}
	if _, err := c.query(ctx, "BEGIN", nil); err != nil {
		return nil, err
	}
	return &transaction{c: c}, nil
}

func (t *transaction) finish(query string) error {
	ctx, cancel := context.WithTimeout(context.Background(), pizzasqlFinishTimeout)
	defer cancel()
	_, err := t.c.query(ctx, query, nil)
	return err
}

func (t *transaction) Commit() error   { return t.finish("COMMIT") }
func (t *transaction) Rollback() error { return t.finish("ROLLBACK") }

func (c *connection) query(ctx context.Context, query string, args []driver.NamedValue) (*pgconn.Result, error) {
	return c.queryResult(ctx, query, args, true)
}

func (c *connection) queryResult(ctx context.Context, query string, args []driver.NamedValue, retryable bool) (*pgconn.Result, error) {
	if c.pg.IsClosed() {
		if retryable {
			return nil, driver.ErrBadConn
		}
		return nil, errors.New("pizzasql: connection is closed")
	}

	sqlText, err := bind(query, args)
	if err != nil {
		return nil, err
	}

	results, err := c.pg.Exec(ctx, sqlText).ReadAll()
	if err != nil {
		if retryable && pgconn.SafeToRetry(err) {
			return nil, driver.ErrBadConn
		}
		return nil, err
	}
	if len(results) != 1 {
		return nil, errors.New("pizzasql: expected one statement result")
	}
	return results[0], results[0].Err
}

func (c *connection) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	r, err := c.query(ctx, query, args)
	if err != nil {
		return nil, err
	}

	res := result{affected: r.CommandTag.RowsAffected()}
	if r.CommandTag.Insert() && res.affected > 0 {
		id, err := c.queryResult(ctx, "SELECT last_insert_rowid()", nil, false)
		if err != nil {
			return nil, fmt.Errorf("pizzasql: insert acknowledged but last insert ID could not be retrieved; the row state is uncertain: %w", err)
		}
		if len(id.Rows) != 1 || len(id.Rows[0]) != 1 || id.Rows[0][0] == nil {
			return nil, errors.New("pizzasql: invalid last insert ID result")
		}
		res.id, err = strconv.ParseInt(string(id.Rows[0][0]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("pizzasql: server did not return an integer insert ID: %w", err)
		}
	}
	return res, nil
}

func (r result) LastInsertId() (int64, error) { return r.id, nil }
func (r result) RowsAffected() (int64, error) { return r.affected, nil }

func (c *connection) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	r, err := c.query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return &rows{fields: r.FieldDescriptions, values: r.Rows}, nil
}

func (r *rows) Columns() []string {
	names := make([]string, len(r.fields))
	for i, f := range r.fields {
		names[i] = f.Name
	}
	return names
}

func (r *rows) Close() error {
	r.values = nil
	return nil
}

func (r *rows) Next(dest []driver.Value) error {
	if r.pos >= len(r.values) {
		return io.EOF
	}

	row := r.values[r.pos]
	for i, v := range row {
		if v == nil {
			dest[i] = nil
			continue
		}
		switch r.fields[i].DataTypeOID {
		case oidTimestamp, oidTimestamptz, oidDate, oidTime:
			t, err := parseTime(string(v))
			if err != nil {
				return err
			}
			dest[i] = t
		default:
			dest[i] = string(v)
		}
	}
	r.pos++
	return nil
}

var timeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
	"15:04:05.999999999",
	"15:04:05",
}

func parseTime(s string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("pizzasql: invalid timestamp value %q", s)
}

// bind inlines the given parameters in place of ? placeholders, quoting each
// value with SQLite syntax so the statement is safe to send over the
// simple-query protocol. Placeholders inside string literals, quoted
// identifiers, and comments are left untouched.
func bind(query string, args []driver.NamedValue) (string, error) {
	var out strings.Builder
	n := 0
	ended := false

	for i := 0; i < len(query); {
		ch := query[i]

		if ch == 0 {
			return "", errors.New("pizzasql: NUL in SQL")
		}

		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			out.WriteByte(ch)
			i++
			continue
		}

		if i+1 < len(query) && query[i:i+2] == "--" {
			end := strings.IndexByte(query[i:], '\n')
			if end < 0 {
				end = len(query) - i
			}
			out.WriteString(query[i : i+end])
			i += end
			continue
		}

		if i+1 < len(query) && query[i:i+2] == "/*" {
			end := strings.Index(query[i+2:], "*/")
			if end < 0 {
				return "", errors.New("pizzasql: unterminated comment")
			}
			end += i + 4
			out.WriteString(query[i:end])
			i = end
			continue
		}

		if ended {
			return "", errors.New("pizzasql: multiple statements are unsupported")
		}

		if ch == ';' {
			ended = true
			out.WriteByte(ch)
			i++
			continue
		}

		if ch == '\'' || ch == '"' || ch == '`' || ch == '[' {
			end, err := scanQuoted(query, i)
			if err != nil {
				return "", err
			}
			out.WriteString(query[i:end])
			i = end
			continue
		}

		if ch == '?' {
			if n >= len(args) {
				return "", errors.New("pizzasql: not enough parameters")
			}
			if args[n].Name != "" {
				return "", errors.New("pizzasql: named parameters are unsupported")
			}
			literal, err := sqlLiteral(args[n].Value)
			if err != nil {
				return "", err
			}
			out.WriteString(literal)
			n++
			i++
			continue
		}

		out.WriteByte(ch)
		i++
	}

	if n != len(args) {
		return "", errors.New("pizzasql: too many parameters")
	}
	return out.String(), nil
}

func scanQuoted(query string, start int) (int, error) {
	delim := query[start]
	end := delim
	escaped := delim == '\''
	if end == '[' {
		end = ']'
	}

	i := start + 1
	for i < len(query) {
		if query[i] != end {
			i++
			continue
		}
		i++
		if escaped && i < len(query) && query[i] == end {
			i++
			continue
		}
		return i, nil
	}
	return 0, errors.New("pizzasql: unterminated quoted token")
}

func sqlLiteral(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "NULL", nil
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return "", errors.New("pizzasql: non-finite number")
		}
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case time.Time:
		return sqlLiteral(v.UTC().Format(time.RFC3339Nano))
	case []byte:
		return "", errors.New("pizzasql: binary parameters are unsupported by the engine")
	case string:
		if strings.ContainsRune(v, 0) {
			return "", errors.New("pizzasql: NUL in text parameter")
		}
		return "'" + strings.ReplaceAll(v, "'", "''") + "'", nil
	default:
		return "", fmt.Errorf("pizzasql: unsupported parameter type %T", value)
	}
}
