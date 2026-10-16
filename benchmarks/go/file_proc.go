package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	filePath := flag.String("file", "", "Path to input file")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Error: --file required")
		os.Exit(1)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	hasher := sha256.New()
	buf := make([]byte, 64*1024)
	var totalBytes int64 = 0

	for {
		n, err := f.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
			hasher.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	digest := hex.EncodeToString(hasher.Sum(nil))
	fmt.Printf("{\"workload\":\"file_proc\",\"bytes_read\":%d,\"sha256\":\"%s\"}\n", totalBytes, digest)
}
