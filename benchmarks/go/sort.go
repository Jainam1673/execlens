package main

import (
	"flag"
	"fmt"
	"sort"
)

func quicksort(arr []int32, low, high int) {
	if low < high {
		p := partition(arr, low, high)
		quicksort(arr, low, p-1)
		quicksort(arr, p+1, high)
	}
}

func partition(arr []int32, low, high int) int {
	// Median of three pivot selection
	mid := low + (high-low)/2
	if arr[mid] < arr[low] {
		arr[low], arr[mid] = arr[mid], arr[low]
	}
	if arr[high] < arr[low] {
		arr[low], arr[high] = arr[high], arr[low]
	}
	if arr[mid] < arr[high] {
		arr[mid], arr[high] = arr[high], arr[mid]
	}

	pivot := arr[high]
	i := low - 1
	for j := low; j < high; j++ {
		if arr[j] <= pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

func main() {
	size := flag.Int("size", 200000, "Array size")
	mode := flag.String("mode", "quicksort", "Sort mode: quicksort or stdlib")
	flag.Parse()

	// Deterministic LCG generation
	arr := make([]int32, *size)
	var state uint64 = 42
	for i := 0; i < *size; i++ {
		state = (state*1664525 + 1013904223) & 0xFFFFFFFF
		arr[i] = int32(state)
	}

	if *mode == "stdlib" {
		sort.Slice(arr, func(i, j int) bool {
			return arr[i] < arr[j]
		})
	} else {
		quicksort(arr, 0, len(arr)-1)
	}

	// Validation
	isSorted := true
	var checksum uint64 = 0
	for i := 0; i < len(arr); i++ {
		if i > 0 && arr[i] < arr[i-1] {
			isSorted = false
		}
		checksum = (checksum*31 + uint64(uint32(arr[i]))) & 0xFFFFFFFFFFFFFFFF
	}

	fmt.Printf("{\"workload\":\"sort\",\"size\":%d,\"mode\":\"%s\",\"sorted\":%t,\"checksum\":\"%x\"}\n",
		*size, *mode, isSorted, checksum)
}
