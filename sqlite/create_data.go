// +build ignore

package main

import (
	"log"
	"math/rand"
	"time"
)

func main() {
	symbols := []string{
		"MSFT",
		"GOOG",
		"AAPL",
		"NVDA",
	}

	db, err := NewTradesDB("trades.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	start := time.Date(2020, time.July, 2, 0, 0, 0, 0, time.UTC)
	delta := 137 * time.Millisecond
	const n = 100_000
	for i := 0; i < n; i++ {
		t := start.Add(time.Duration(i) * delta)
		tr := Trade{
			Time:   t,
			Symbol: symbols[rand.Intn(len(symbols))],
			Price:  rand.Float64() * 500.0,
			IsBuy:  rand.Intn(2) == 0,
		}
		if err := db.AddTrade(tr); err != nil {
			log.Fatal(err)
		}
	}
}
