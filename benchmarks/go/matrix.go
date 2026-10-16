package main

import (
	"flag"
	"fmt"
)

func main() {
	dim := flag.Int("dim", 300, "Matrix dimension N")
	flag.Parse()

	n := *dim
	a := make([]float64, n*n)
	b := make([]float64, n*n)
	c := make([]float64, n*n)

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			a[i*n+j] = float64((i*31+j*17)%100) / 10.0
			b[i*n+j] = float64((i*13+j*23)%100) / 10.0
		}
	}

	// Matrix multiplication: C = A * B
	for i := 0; i < n; i++ {
		for k := 0; k < n; k++ {
			aik := a[i*n+k]
			for j := 0; j < n; j++ {
				c[i*n+j] += aik * b[k*n+j]
			}
		}
	}

	var checksum float64 = 0
	for i := 0; i < n*n; i++ {
		checksum += c[i]
	}

	fmt.Printf("{\"workload\":\"matrix\",\"dimension\":%d,\"checksum\":\"%.2f\"}\n", n, checksum)
}
