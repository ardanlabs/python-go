// Package trades provides an SQLite based trades database.
package trades

import (
	"database/sql"
	"errors"
	"fmt"
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

// Trade is a buy/sell trade for symbol.
type Trade struct {
	Time   time.Time
	Symbol string
	Price  float64
	IsBuy  bool
}

// DB is a database of stock trades.
type DB struct {
	sql    *sql.DB
	stmt   *sql.Stmt
	buffer []Trade
}

// NewDB constructs a Trades value for managing stock trades in a
// SQLite database. This API is not thread safe.
func NewDB(dbFile string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	if _, err = sqlDB.Exec(schemaSQL); err != nil {
		return nil, err
	}

	stmt, err := sqlDB.Prepare(insertSQL)
	if err != nil {
		return nil, err
	}

	db := DB{
		sql:    sqlDB,
		stmt:   stmt,
		buffer: make([]Trade, 0, 1024),
	}
	return &db, nil
}

// Add stores a trade into the buffer. Once the buffer is full, the
// trades are flushed to the database.
func (db *DB) Add(trade Trade) error {
	if len(db.buffer) == cap(db.buffer) {
		return errors.New("trades buffer is full")
	}

	db.buffer = append(db.buffer, trade)
	if len(db.buffer) == cap(db.buffer) {
		if err := db.Flush(); err != nil {
			return fmt.Errorf("unable to flush trades: %w", err)
		}
	}

	return nil
}

// Flush inserts pending trades into the database.
func (db *DB) Flush() error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}

	for _, trade := range db.buffer {
		_, err := tx.Stmt(db.stmt).Exec(trade.Time, trade.Symbol, trade.Price, trade.IsBuy)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	db.buffer = db.buffer[:0]
	return tx.Commit()
}

// Close flushes all trades to the database and prevents any future trading.
func (db *DB) Close() (err error) {
	defer func() {
		if cerr := db.sql.Close(); cerr != nil {
			err = cerr
		}
	}()

	defer func() {
		if serr := db.stmt.Close(); serr != nil {
			err = serr
		}
	}()

	if err := db.Flush(); err != nil {
		return err
	}

	return nil
}
