package main

import (
	"flag"
	"fmt"
)

func main() {
	limit := flag.Int("limit", 5000000, "Prime limit")
	flag.Parse()

	if *limit < 2 {
		fmt.Printf("{\"workload\":\"sieve\",\"limit\":%d,\"count\":0,\"checksum\":\"0\"}\n", *limit)
		return
	}

	// 1 byte per candidate for algorithmic parity
	sieve := make([]byte, *limit+1)
	count := 0

	for p := 2; p*p <= *limit; p++ {
		if sieve[p] == 0 {
			for i := p * p; i <= *limit; i += p {
				sieve[i] = 1
			}
		}
	}

	for p := 2; p <= *limit; p++ {
		if sieve[p] == 0 {
			count++
		}
	}

	fmt.Printf("{\"workload\":\"sieve\",\"limit\":%d,\"count\":%d,\"checksum\":\"%d\"}\n", *limit, count, count)
}
