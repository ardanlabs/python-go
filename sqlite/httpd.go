package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var (
	db *TradesDB
)

func tradeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()

	var tr Trade
	if err := json.NewDecoder(r.Body).Decode(&tr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.AddTrade(tr); err != nil {
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
