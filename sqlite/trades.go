// Package trades provides an SQLite based trades database
package trades

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	insertSQL = `
INSERT INTO trades (
	time, symbol, price, buy
) VALUES (
	?, ?, ?, ?
)
`

	schemaSQL = `
CREATE TABLE IF NOT EXISTS trades (
    time TIMESTAMP,
    symbol VARCHAR(32),
    price FLOAT,
    buy BOOLEAN
);

CREATE INDEX IF NOT EXISTS trades_time ON trades(time);
CREATE INDEX IF NOT EXISTS trades_symbol ON trades(symbol);
`
)

// Trade is a buy/sell trade for symbol
type Trade struct {
	Time   time.Time
	Symbol string
	Price  float64
	IsBuy  bool
}

// DB is a database of stocks
type DB struct {
	db     *sql.DB
	stmt   *sql.Stmt
	buffer []Trade
}

// NewDB connect to SQLite database in dbFile
// Tables will be created if they don't exist
// The returned DB is not goroutine safe
func NewDB(dbFile string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(schemaSQL)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return nil, err
	}

	tdb := &DB{
		db:     db,
		stmt:   stmt,
		buffer: make([]Trade, 0, 1024),
	}
	return tdb, nil
}

// Close closes all database related resources
func (db *DB) Close() error {
	// TODO: Wrap errors
	db.Flush()
	db.stmt.Close()
	return db.db.Close()
}

// Flush inserts pending trades into the database
func (db *DB) Flush() error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	for _, t := range db.buffer {
		_, err := tx.Stmt(db.stmt).Exec(t.Time, t.Symbol, t.Price, t.IsBuy)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	db.buffer = db.buffer[:0]
	return tx.Commit()
}

// AddTrade adds a new trade.
// The new trade is only added to the internal buffer and will be inserted
// to the database later
func (db *DB) AddTrade(t Trade) error {
	// TODO: We might grow indefinitely on persistent Flush errors
	db.buffer = append(db.buffer, t)
	if len(db.buffer) == cap(db.buffer) {
		return db.Flush()
	}
	return nil
}
