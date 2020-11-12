package trades

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	require := require.New(t)

	db, err := NewDB("/tmp/trades.db")
	require.NoError(err)
	defer db.Close()

	tr := Trade{
		Time:   time.Now(),
		Symbol: "MSFT",
		Price:  216.39,
		IsBuy:  false,
	}

	err = db.AddTrade(tr)
	require.NoError(err)

	// TODO: Check database
}

func BenchmarkAdd(b *testing.B) {
	b.StopTimer()
	require := require.New(b)
	file, err := ioutil.TempFile("", "*.db")
	require.NoError(err)
	file.Close()
	b.Logf("db file: %s", file.Name())
	db, err := NewDB(file.Name())
	require.NoError(err)
	tr := Trade{
		Time:   time.Now(),
		Symbol: "MSFT",
		Price:  216.39,
		IsBuy:  false,
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		err := db.AddTrade(tr)
		require.NoError(err)
	}
}
