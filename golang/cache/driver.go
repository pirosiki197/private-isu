package cache

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/motoki317/sc"
	"github.com/traP-jp/h24w-17/domains"
	"github.com/traP-jp/h24w-17/normalizer"
)

var queryMap = make(map[string]domains.CachePlanQuery)

var tableSchema = make(map[string]domains.TableSchema)

const cachePlanRaw = `queries:
  - query: DELETE FROM ` + "`" + `users` + "`" + ` WHERE ` + "`" + `id` + "`" + ` > 1000;
    type: delete
    table: users
  - query: SELECT COUNT(*) AS ` + "`" + `count` + "`" + ` FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `post_id` + "`" + ` IN (?);
    type: select
    table: comments
    cache: true
    conditions:
      - column: post_id
        operator: in
        placeholder:
          index: 0
  - query: SELECT COUNT(*) AS ` + "`" + `count` + "`" + ` FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `post_id` + "`" + ` = ?;
    type: select
    table: comments
    cache: true
    conditions:
      - column: post_id
        operator: eq
        placeholder:
          index: 0
  - query: UPDATE ` + "`" + `users` + "`" + ` SET ` + "`" + `del_flg` + "`" + ` = 1 WHERE ` + "`" + `id` + "`" + ` % 50 = 0;
    type: update
    table: users
  - query: SELECT ` + "`" + `id` + "`" + `, ` + "`" + `user_id` + "`" + `, ` + "`" + `body` + "`" + `, ` + "`" + `mime` + "`" + `, ` + "`" + `created_at` + "`" + ` FROM ` + "`" + `posts` + "`" + ` WHERE ` + "`" + `user_id` + "`" + ` = ? ORDER BY ` + "`" + `created_at` + "`" + ` DESC;
    type: select
    table: posts
    cache: true
    conditions:
      - column: user_id
        operator: eq
        placeholder:
          index: 0
  - query: INSERT INTO ` + "`" + `comments` + "`" + ` (` + "`" + `post_id` + "`" + `, ` + "`" + `user_id` + "`" + `, ` + "`" + `comment` + "`" + `) VALUES (?);
    type: insert
    table: comments
    columns:
      - post_id
      - user_id
      - comment
  - query: UPDATE ` + "`" + `users` + "`" + ` SET ` + "`" + `del_flg` + "`" + ` = 0;
    type: update
    table: users
  - query: SELECT * FROM ` + "`" + `users` + "`" + ` WHERE ` + "`" + `id` + "`" + ` = ?;
    type: select
    table: users
    cache: true
    conditions:
      - column: id
        operator: eq
        placeholder:
          index: 0
  - query: SELECT * FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `post_id` + "`" + ` = ? ORDER BY ` + "`" + `created_at` + "`" + ` DESC;
    type: select
    table: comments
    cache: true
    conditions:
      - column: post_id
        operator: eq
        placeholder:
          index: 0
  - query: SELECT ` + "`" + `id` + "`" + `, ` + "`" + `user_id` + "`" + `, ` + "`" + `body` + "`" + `, ` + "`" + `mime` + "`" + `, ` + "`" + `created_at` + "`" + ` FROM ` + "`" + `posts` + "`" + ` WHERE ` + "`" + `created_at` + "`" + ` <= ? ORDER BY ` + "`" + `created_at` + "`" + ` DESC;
    type: select
    table: posts
    cache: false
  - query: INSERT INTO ` + "`" + `posts` + "`" + ` (` + "`" + `user_id` + "`" + `, ` + "`" + `mime` + "`" + `, ` + "`" + `imgdata` + "`" + `, ` + "`" + `body` + "`" + `) VALUES (?);
    type: insert
    table: posts
    columns:
      - user_id
      - mime
      - imgdata
      - body
  - query: SELECT 1 FROM ` + "`" + `users` + "`" + ` WHERE ` + "`" + `account_name` + "`" + ` = ?;
    type: select
    table: users
    cache: true
    conditions:
      - column: account_name
        operator: eq
        placeholder:
          index: 0
  - query: SELECT ` + "`" + `id` + "`" + `, ` + "`" + `user_id` + "`" + `, ` + "`" + `body` + "`" + `, ` + "`" + `mime` + "`" + `, ` + "`" + `created_at` + "`" + ` FROM ` + "`" + `posts` + "`" + ` ORDER BY ` + "`" + `created_at` + "`" + ` DESC;
    type: select
    table: posts
    cache: true
  - query: SELECT COUNT(*) AS ` + "`" + `count` + "`" + ` FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `post_id` + "`" + ` = ?;
    type: select
    table: comments
    cache: true
    conditions:
      - column: post_id
        operator: eq
        placeholder:
          index: 0
  - query: INSERT INTO ` + "`" + `users` + "`" + ` (` + "`" + `account_name` + "`" + `, ` + "`" + `passhash` + "`" + `) VALUES (?);
    type: insert
    table: users
    columns:
      - account_name
      - passhash
  - query: SELECT * FROM ` + "`" + `users` + "`" + ` WHERE ` + "`" + `account_name` + "`" + ` = ? AND ` + "`" + `del_flg` + "`" + ` = 0;
    type: select
    table: users
    cache: true
    conditions:
      - column: account_name
        operator: eq
        placeholder:
          index: 0
  - query: SELECT * FROM ` + "`" + `posts` + "`" + ` WHERE ` + "`" + `id` + "`" + ` = ?;
    type: select
    table: posts
    cache: true
    conditions:
      - column: id
        operator: eq
        placeholder:
          index: 0
  - query: DELETE FROM ` + "`" + `posts` + "`" + ` WHERE ` + "`" + `id` + "`" + ` > 10000;
    type: delete
    table: posts
  - query: DELETE FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `id` + "`" + ` > 100000;
    type: delete
    table: comments
  - query: SELECT * FROM ` + "`" + `comments` + "`" + ` WHERE ` + "`" + `post_id` + "`" + ` = ? ORDER BY ` + "`" + `created_at` + "`" + ` DESC LIMIT 3;
    type: select
    table: comments
    cache: true
    conditions:
      - column: post_id
        operator: eq
        placeholder:
          index: 0
  - query: SELECT ` + "`" + `id` + "`" + ` FROM ` + "`" + `posts` + "`" + ` WHERE ` + "`" + `user_id` + "`" + ` = ?;
    type: select
    table: posts
    cache: true
    conditions:
      - column: user_id
        operator: eq
        placeholder:
          index: 0
`

const schemaRaw = `-- benchmarker/userdata/load.rbから読み込まれる

DROP TABLE IF EXISTS users;
CREATE TABLE users (
  ` + "`" + `id` + "`" + ` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  ` + "`" + `account_name` + "`" + ` varchar(64) NOT NULL UNIQUE,
  ` + "`" + `passhash` + "`" + ` varchar(128) NOT NULL, -- SHA2 512 non-binary (hex)
  ` + "`" + `authority` + "`" + ` tinyint(1) NOT NULL DEFAULT 0,
  ` + "`" + `del_flg` + "`" + ` tinyint(1) NOT NULL DEFAULT 0,
  ` + "`" + `created_at` + "`" + ` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS posts;
CREATE TABLE posts (
  ` + "`" + `id` + "`" + ` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  ` + "`" + `user_id` + "`" + ` int NOT NULL,
  ` + "`" + `mime` + "`" + ` varchar(64) NOT NULL,
  ` + "`" + `imgdata` + "`" + ` mediumblob NOT NULL,
  ` + "`" + `body` + "`" + ` text NOT NULL,
  ` + "`" + `created_at` + "`" + ` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS comments;
CREATE TABLE comments (
  ` + "`" + `id` + "`" + ` int NOT NULL AUTO_INCREMENT PRIMARY KEY,
  ` + "`" + `post_id` + "`" + ` int NOT NULL,
  ` + "`" + `user_id` + "`" + ` int NOT NULL,
  ` + "`" + `comment` + "`" + ` text NOT NULL,
  ` + "`" + `created_at` + "`" + ` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;
`

func init() {
	sql.Register("mysql+cache", CacheDriver{})

	schema, err := domains.LoadTableSchema(schemaRaw)
	if err != nil {
		panic(err)
	}
	for _, table := range schema {
		tableSchema[table.TableName] = table
	}

	plan, err := domains.LoadCachePlan(strings.NewReader(cachePlanRaw))
	if err != nil {
		panic(err)
	}

	for _, query := range plan.Queries {
		queryMap[query.Query] = *query
		if query.Type != domains.CachePlanQueryType_SELECT {
			continue
		}

		if query.Select.Cache {
			conditions := query.Select.Conditions
			if len(conditions) == 1 {
				condition := conditions[0]
				column := tableSchema[query.Select.Table].Columns[condition.Column]
				if (column.IsPrimary || column.IsUnique) && condition.Operator == domains.CachePlanOperator_EQ {
					caches[query.Query] = cacheWithInfo{
						query:      query.Query,
						info:       *query.Select,
						cache:      sc.NewMust(replaceFn, 10*time.Minute, 10*time.Minute),
						uniqueOnly: true,
					}
					continue
				}
			}
			caches[query.Query] = cacheWithInfo{
				query:      query.Query,
				info:       *query.Select,
				cache:      sc.NewMust(replaceFn, 10*time.Minute, 10*time.Minute),
				uniqueOnly: false,
			}
		}

		// TODO: if query is like "SELECT * FROM WHERE pk IN (?, ?, ...)", generate cache with query "SELECT * FROM table WHERE pk = ?"
	}

	for _, cache := range caches {
		cacheByTable[cache.info.Table] = append(cacheByTable[cache.info.Table], cache)
	}
}

var _ driver.Driver = CacheDriver{}

type CacheDriver struct{}

func (d CacheDriver) Open(dsn string) (driver.Conn, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	c, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := c.Connect(context.Background())
	if err != nil {
		return nil, err
	}
	return &cacheConn{inner: conn}, nil
}

var (
	_ driver.Conn           = &cacheConn{}
	_ driver.ConnBeginTx    = &cacheConn{}
	_ driver.Pinger         = &cacheConn{}
	_ driver.QueryerContext = &cacheConn{}
)

type cacheConn struct {
	inner driver.Conn
}

func (c *cacheConn) Prepare(rawQuery string) (driver.Stmt, error) {
	normalizedQuery := normalizer.NormalizeQuery(rawQuery)

	queryInfo, ok := queryMap[normalizedQuery]
	if !ok {
		return c.inner.Prepare(rawQuery)
	}

	if queryInfo.Type == domains.CachePlanQueryType_SELECT && !queryInfo.Select.Cache {
		return c.inner.Prepare(rawQuery)
	}

	innerStmt, err := c.inner.Prepare(rawQuery)
	if err != nil {
		return nil, err
	}
	return &customCacheStatement{
		inner:     innerStmt,
		conn:      c,
		rawQuery:  rawQuery,
		query:     normalizedQuery,
		queryInfo: queryInfo,
	}, nil
}

func (c *cacheConn) Close() error {
	return c.inner.Close()
}

func (c *cacheConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *cacheConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if i, ok := c.inner.(driver.ConnBeginTx); ok {
		return i.BeginTx(ctx, opts)
	}
	return c.inner.Begin()
}

func (c *cacheConn) Ping(ctx context.Context) error {
	if i, ok := c.inner.(driver.Pinger); ok {
		return i.Ping(ctx)
	}
	return nil
}

var _ driver.Rows = &cacheRows{}

type cacheRows struct {
	inner   driver.Rows
	cached  bool
	columns []string
	rows    sliceRows
	limit   int

	mu sync.Mutex
}

func (r *cacheRows) Clone() *cacheRows {
	if !r.cached {
		panic("cannot clone uncached rows")
	}
	return &cacheRows{
		inner:   r.inner,
		cached:  r.cached,
		columns: r.columns,
		rows:    r.rows.clone(),
		limit:   r.limit,
	}
}

func newCacheRows(inner driver.Rows) *cacheRows {
	return &cacheRows{inner: inner}
}

type row = []driver.Value

type sliceRows struct {
	rows []row
	idx  int
}

func (r sliceRows) clone() sliceRows {
	rows := make([]row, len(r.rows))
	copy(rows, r.rows)
	return sliceRows{rows: rows}
}

func (r *sliceRows) append(row ...row) {
	r.rows = append(r.rows, row...)
}

func (r *sliceRows) reset() {
	r.idx = 0
}

func (r *sliceRows) Next(dest []driver.Value, limit int) error {
	if r.idx >= len(r.rows) {
		r.reset()
		return io.EOF
	}
	if limit > 0 && r.idx >= limit {
		r.reset()
		return io.EOF
	}
	row := r.rows[r.idx]
	r.idx++
	copy(dest, row)
	return nil
}

func (r *cacheRows) Columns() []string {
	if r.cached {
		return r.columns
	}
	columns := r.inner.Columns()
	r.columns = make([]string, len(columns))
	copy(r.columns, columns)
	return columns
}

func (r *cacheRows) Close() error {
	if r.cached {
		r.rows.reset()
		return nil
	}
	return r.inner.Close()
}

func (r *cacheRows) Next(dest []driver.Value) error {
	if r.cached {
		return r.rows.Next(dest, r.limit)
	}

	err := r.inner.Next(dest)
	if err != nil {
		if err == io.EOF {
			r.cached = true
			return err
		}
		return err
	}

	cachedRow := make(row, len(dest))
	for i := 0; i < len(dest); i++ {
		switch v := dest[i].(type) {
		case int64, uint64, float64, string, bool, time.Time, nil: // no need to copy
			cachedRow[i] = v
		case []byte: // copy to prevent mutation
			data := make([]byte, len(v))
			copy(data, v)
			cachedRow[i] = data
		default:
			// TODO: handle other types
			// Should we mark this row as uncacheable?
		}
	}
	r.rows.append(cachedRow)

	return nil
}

func mergeCachedRows(rows []*cacheRows) *cacheRows {
	if len(rows) == 0 {
		return nil
	}
	if len(rows) == 1 {
		return rows[0]
	}

	mergedSlice := sliceRows{}
	for _, r := range rows {
		mergedSlice.append(r.rows.rows...)
	}

	return &cacheRows{
		cached:  true,
		columns: rows[0].columns,
		rows:    mergedSlice,
	}
}

func (r *cacheRows) createCache() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	columns := r.Columns()
	dest := make([]driver.Value, len(columns))
	for {
		err := r.Next(dest)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	r.Close()
	return nil
}
