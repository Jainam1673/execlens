package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Record struct {
	ID     int      `json:"id"`
	User   string   `json:"user"`
	Amount float64  `json:"amount"`
	Active bool     `json:"active"`
	Tags   []string `json:"tags"`
}

func main() {
	inputPath := flag.String("input", "", "Path to input JSON file")
	flag.Parse()

	if *inputPath == "" {
		fmt.Println("Error: --input required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*inputPath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	activeCount := 0
	totalAmount := 0.0

	for _, r := range records {
		if r.Active {
			activeCount++
			totalAmount += r.Amount
		}
	}

	fmt.Printf("{\"workload\":\"json_proc\",\"active_count\":%d,\"total_amount\":%.2f}\n", activeCount, totalAmount)
}
