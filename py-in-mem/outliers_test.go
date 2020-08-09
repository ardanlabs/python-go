package outliers

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	require := require.New(t)
	require.NoError(Initialize())

	o, err := NewOutliers("outliers", "detect")
	require.NoError(err, "new")

	const size = 1000
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Float64()
	}

	data[7] = 97.3
	data[113] = 92.1
	data[835] = 93.2

	out, err := o.Detect(data)
	require.NoError(err, "detect")
	require.Equal([]int{7, 113, 835}, out, "outliers")
}

func TestNotFound(t *testing.T) {
	require := require.New(t)
	require.NoError(Initialize())

	_, err := NewOutliers("outliers", "no-such-function")
	require.Error(err, "attribute")

	_, err = NewOutliers("no_such_module", "detect")
	require.Error(err, "module")
}
