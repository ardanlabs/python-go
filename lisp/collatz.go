package main

import (
	"fmt"
)

func collatzStep(n int) int {
	if n%2 == 0 {
		return n / 2
	}
	return n*3 + 1
}

func main() {
	fmt.Println(collatzStep(7)) // 22
}
