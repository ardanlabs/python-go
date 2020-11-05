package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
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

type Trade struct {
	Time   time.Time
	Symbol string
	Price  float64
	IsBuy  bool `json:"buy"`
}

// TradesDB is a database of stocks
type TradesDB struct {
	db   *sql.DB
	stmt *sql.Stmt
}

func NewTradesDB(dbFile string) (*TradesDB, error) {
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

	return &TradesDB{db, stmt}, nil
}

func (db *TradesDB) Close() error {
	return db.db.Close()
}

func (db *TradesDB) AddTrade(t Trade) error {
	// TODO: Batching
	_, err := db.stmt.Exec(t.Time, t.Symbol, t.Price, t.IsBuy)
	return err
}
