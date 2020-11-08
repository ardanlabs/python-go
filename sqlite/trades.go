package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
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

	buffer := make([]Trade, 0, 1024)
	return &TradesDB{db, stmt, buffer}, nil
}

func (db *TradesDB) Close() error {
	// TODO: Check errors from Flush & stmt.Close
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

func (db *TradesDB) AddTrade(t Trade) error {
	// FIXME: We might grow buffer indefinitely on consistent Flush errors
	db.buffer = append(db.buffer, t)
	if len(db.buffer) == cap(db.buffer) {
		if err := db.Flush(); err != nil {
			return err
		}
	}
	return nil
}

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
	log.Printf("conneted to %s", dbFile)

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
