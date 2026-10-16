package main

import (
	"flag"
	"fmt"
)

func main() {
	operations := flag.Int("operations", 300000, "Number of map operations")
	flag.Parse()

	m := make(map[string]int64, *operations/2)

	// Insertion phase
	for i := 0; i < *operations; i++ {
		key := fmt.Sprintf("k_%d", i%(*operations/2))
		m[key] = int64(i * 3)
	}

	// Lookup phase with mixed hits and misses
	var checksum int64 = 0
	for i := 0; i < *operations; i++ {
		key := fmt.Sprintf("k_%d", i)
		if val, exists := m[key]; exists {
			checksum += val
		} else {
			checksum -= 1
		}
	}

	fmt.Printf("{\"workload\":\"hashmap\",\"operations\":%d,\"checksum\":\"%d\"}\n", *operations, checksum)
}
