package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db *TradesDB

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
	IsBuy  bool `json:"buy"`
}

func newBuffer() []Trade {
	return make([]Trade, 0, 1024)
}

// TradesDB is a database of stocks
type TradesDB struct {
	db     *sql.DB
	stmt   *sql.Stmt
	buffer []Trade
	m      sync.Mutex
}

// NewTradesDB connect to SQLite database in dbFile
// Tables will be created if they don't exist
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

	tdb := &TradesDB{
		db:     db,
		stmt:   stmt,
		buffer: newBuffer(),
	}
	return tdb, nil
}

// Close closes all database related resources
func (db *TradesDB) Close() error {
	db.m.Lock()
	pending := db.buffer
	db.buffer = nil
	db.m.Unlock()

	// TODO: Check errors from Flush & stmt.Close
	db.insert(pending)
	db.stmt.Close()
	return db.db.Close()
}

// insert inserts pending trades into the database
func (db *TradesDB) insert(trades []Trade) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}

	for _, t := range trades {
		_, err := tx.Stmt(db.stmt).Exec(t.Time, t.Symbol, t.Price, t.IsBuy)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (db *TradesDB) add(t Trade) []Trade {
	db.m.Lock()
	defer db.m.Unlock()

	db.buffer = append(db.buffer, t)
	if len(db.buffer) < cap(db.buffer) {
		return nil
	}

	// Reset buffer and return trades to insert
	pending := db.buffer
	db.buffer = newBuffer()
	return pending
}

// AddTrade adds a new trade.
// The new trade is only added to the internal buffer and will be inserted
// to the database later
func (db *TradesDB) AddTrade(t Trade) error {
	pending := db.add(t)
	if pending == nil {
		return nil
	}

	return db.insert(pending)
}

// tradeHandler handles a new trade notification
func tradeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}

	if db == nil {
		log.Printf("DB uninitialized")
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()

	var tr Trade
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		log.Printf("json decode error: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.AddTrade(tr); err != nil {
		log.Printf("add error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	dbFile := os.Getenv("DB_FILE")
	if dbFile == "" {
		dbFile = "trades.db"
	}

	var err error
	db, err = NewTradesDB(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("connected to %s", dbFile)

	http.HandleFunc("/trade", tradeHandler)

	addr := os.Getenv("HTTPD_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
