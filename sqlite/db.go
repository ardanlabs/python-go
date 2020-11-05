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
	db     *sql.DB
	stmt   *sql.Stmt
	buffer []Trade
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

	buffer := make([]Trade, 0, 1024)
	return &TradesDB{db, stmt, buffer}, nil
}

func (db *TradesDB) Close() error {
	db.Flush()
	db.stmt.Close()
	return db.db.Close()
}

func (db *TradesDB) Flush() error {
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
	err = tx.Commit()
	if err == nil {
		db.buffer = db.buffer[:0]
	}
	return err
}

func (db *TradesDB) bufferFull() bool {
	return len(db.buffer) == cap(db.buffer)
}

func (db *TradesDB) AddTrade(t Trade) error {
	// FIXME: We might grow buffer indefinitely on consistent Flush errors
	db.buffer = append(db.buffer, t)
	if db.bufferFull() {
		if err := db.Flush(); err != nil {
			return err
		}
	}
	return nil
}
