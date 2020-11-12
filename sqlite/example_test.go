package trades_test

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ardanlabs/python-go/trades"
)

func ExampleDB() {
	dbFile := "/tmp/db-test" + time.Now().Format(time.RFC3339)
	db, err := trades.NewDB(dbFile)
	if err != nil {
		fmt.Println("ERROR: create -", err)
		return
	}
	defer db.Close()

	const count = 10_000
	for i := 0; i < count; i++ {
		t := trades.Trade{
			Time:   time.Now(),
			Symbol: "AAPL",
			Price:  rand.Float64() * 200,
			IsBuy:  i%2 == 0,
		}
		if err := db.AddTrade(t); err != nil {
			fmt.Println("ERROR: insert - ", err)
			return
		}
	}

	fmt.Printf("inserted %d records\n", count)
	// Output:
	// inserted 10000 records
}
