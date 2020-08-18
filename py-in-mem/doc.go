/* Package outliers provides outlier detection by calling a Python function.

You *must* have numpy installed and the Python function you're calling should
be importable (in the PYTHONPATH).

You need to set CGO_CFLAGS before building the code

	$ export CGO_CFLAGS="-I $(python -c 'import numpy; print(numpy.get_include())'"
	$ go build

Example:

import (
	"github.com/ardanlabs/python-go/outliers"
	"fmt"
)

func main() {
	// Create data
	const size = 1000
	data := make([]float64, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Float64()
	}
	data[9] = 92.3
	data[238] = 103.2
	data[743] = 86.1

	// Use "detect" function from "outliers" module
	o, err := outliers.NewOutliers("outliers", "detect")
	if err != nil {
		fmt.Printf("can't load 'outliers.detect': %s", err)
		return
	}
	indices, err := o.Detect(data)
	if err != nil {
		fmt.Println("can't call outliers.detect: %s", err)
		return
	}
	fmt.Println(indices) // [9 238 743]
	o.Close() // Free the underlying Python function
}
*/
package outliers
