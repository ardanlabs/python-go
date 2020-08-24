// +build ignore

package main

import (
	"fmt"
)

func genData() []float64 {
	return nil
}

func Example() error {
	data := genData()
	o, err := NewOutliers("outliers", "detect")
	if err != nil {
		return err
	}
	defer o.Close()
	indices, err := o.Detect(data)
	if err != nil {
		return err
	}
	fmt.Printf("outliers at: %v\n", indices)
	return nil
}
