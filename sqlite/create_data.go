package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/ardanlabs/python-go/sqlite/trades"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	symbols := []string{
		"MSFT",
		"GOOG",
		"AAPL",
		"NVDA",
	}

	db, err := trades.NewDB("trades.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	start := time.Date(2020, time.July, 2, 0, 0, 0, 0, time.UTC)
	delta := 137 * time.Millisecond

	for i := 0; i < 100_000; i++ {
		time := start.Add(time.Duration(i) * delta)
		trade := trades.Trade{
			Time:   time,
			Symbol: symbols[rand.Intn(len(symbols))],
			Price:  rand.Float64() * 500.0,
			IsBuy:  rand.Intn(2) == 0,
		}
		if err := db.Add(trade); err != nil {
			log.Fatal(err)
		}
	}
}
