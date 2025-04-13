package main

import (
	"fmt"
	"os"

	"github.com/go-to-k/go-speed-chan-vs-goroutine/benchmark"
)

func main() {
	if err := benchmark.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
