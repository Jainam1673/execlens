package workloads

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// WorkloadParam defines a configurable parameter for a benchmark workload.
type WorkloadParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "int", "string", "bool"
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
}

// WorkloadContract defines the explicit specification for a benchmark workload.
type WorkloadContract struct {
	ID                 string                   `json:"id"`
	Version            string                   `json:"version"`
	Name               string                   `json:"name"`
	Description        string                   `json:"description"`
	AlgorithmPurpose   string                   `json:"algorithm_purpose"`
	Parameters         []WorkloadParam          `json:"parameters"`
	SupportedModes     []string                 `json:"supported_modes"`
	FairnessCaveats    string                   `json:"fairness_caveats"`
	OperationsCount    func(params map[string]interface{}) int64                   `json:"-"`
	TotalBytesCount    func(params map[string]interface{}) int64                   `json:"-"`
	ValidateResult     func(params map[string]interface{}, stdout string) (bool, string, string) `json:"-"`
	PrepareDataset     func(params map[string]interface{}, tempDir string) (datasetPath string, err error) `json:"-"`
	ToCLIArgs          func(params map[string]interface{}, datasetPath string) []string `json:"-"`
}

// Sieve Primes count helper for validation
func countPrimesExpected(limit int) int {
	if limit < 2 {
		return 0
	}
	sieve := make([]bool, limit+1)
	count := 0
	for p := 2; p*p <= limit; p++ {
		if !sieve[p] {
			for i := p * p; i <= limit; i += p {
				sieve[i] = true
			}
		}
	}
	for p := 2; p <= limit; p++ {
		if !sieve[p] {
			count++
		}
	}
	return count
}

// Deterministic LCG for reproducible pseudo-random integer generation across all languages:
// state = (state * 1664525 + 1013904223) & 0xFFFFFFFF
type LCG struct {
	state uint64
}

func NewLCG(seed uint64) *LCG {
	return &LCG{state: seed}
}

func (l *LCG) Next() uint32 {
	l.state = (l.state*1664525 + 1013904223) & 0xFFFFFFFF
	return uint32(l.state)
}

// Registry manages the catalog of available workloads.
type Registry struct {
	workloads map[string]*WorkloadContract
	ordered   []string
}

func NewRegistry() *Registry {
	r := &Registry{
		workloads: make(map[string]*WorkloadContract),
	}

	// 1. Sieve of Eratosthenes
	r.Register(&WorkloadContract{
		ID:               "sieve",
		Version:          "1.0.0",
		Name:             "Sieve of Eratosthenes",
		Description:      "Calculates the count of prime numbers up to a specified limit using the Sieve of Eratosthenes algorithm.",
		AlgorithmPurpose: "Measures raw CPU compute throughput, branch prediction efficiency, and memory bandwidth in linear boolean array traversal.",
		Parameters: []WorkloadParam{
			{Name: "limit", Type: "int", Default: 5000000, Description: "Upper integer bound for prime sieve"},
		},
		SupportedModes:  []string{"standard"},
		FairnessCaveats: "Language implementations utilize boolean / 8-bit byte arrays. Memory layout differences (e.g. std::vector<bool> bitset vs byte array) can alter cache utilization; all implementations here allocate 1 byte per candidate for algorithmic parity.",
		OperationsCount: func(params map[string]interface{}) int64 {
			limit := getIntParam(params, "limit", 5000000)
			return int64(limit)
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			return 0
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			limit := getIntParam(params, "limit", 5000000)
			expectedPrimes := countPrimesExpected(limit)
			expectedStr := fmt.Sprintf("%d", expectedPrimes)

			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, fmt.Sprintf("Expected JSON output containing count %d", expectedPrimes)
			}
			countVal, ok := outMap["count"]
			if !ok {
				return false, stdout, fmt.Sprintf("Missing 'count' field in JSON. Expected %d", expectedPrimes)
			}
			actualCount := int(toNumeric(countVal))
			if actualCount != expectedPrimes {
				return false, fmt.Sprintf("%d", actualCount), expectedStr
			}
			return true, fmt.Sprintf("%d primes", actualCount), expectedStr
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			limit := getIntParam(params, "limit", 5000000)
			return []string{"--limit", fmt.Sprintf("%d", limit)}
		},
	})

	// 2. Integer Sorting
	r.Register(&WorkloadContract{
		ID:               "sort",
		Version:          "1.0.0",
		Name:             "Integer Sorting",
		Description:      "Sorts a deterministically generated 32-bit integer array and validates the non-decreasing order and checksum.",
		AlgorithmPurpose: "Measures array indexing, memory cache locality, and comparison branching.",
		Parameters: []WorkloadParam{
			{Name: "size", Type: "int", Default: 200000, Description: "Number of integers to generate and sort"},
			{Name: "mode", Type: "string", Default: "quicksort", Description: "Sort implementation: 'quicksort' (equivalent algorithm) or 'stdlib' (standard library sort)"},
		},
		SupportedModes:  []string{"quicksort", "stdlib"},
		FairnessCaveats: "'quicksort' compares strictly identical in-place median-of-three quicksort implementations. 'stdlib' compares the runtime's standard sort (e.g. pdqsort, timsort, introsort) and should not be cited as pure language execution speed.",
		OperationsCount: func(params map[string]interface{}) int64 {
			size := getIntParam(params, "size", 200000)
			return int64(size)
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			return 0
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, "Expected JSON output with sorted=true and valid checksum"
			}
			sorted, _ := outMap["sorted"].(bool)
			checksum, _ := outMap["checksum"].(string)
			if !sorted || checksum == "" {
				return false, fmt.Sprintf("sorted=%v, checksum=%s", sorted, checksum), "sorted=true with non-empty checksum"
			}
			return true, fmt.Sprintf("sorted=true, checksum=%s", checksum), "sorted=true"
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			size := getIntParam(params, "size", 200000)
			mode := getStringParam(params, "mode", "quicksort")
			return []string{"--size", fmt.Sprintf("%d", size), "--mode", mode}
		},
	})

	// 3. Hash Map Operations
	r.Register(&WorkloadContract{
		ID:               "hashmap",
		Version:          "1.0.0",
		Name:             "Hash Map Operations",
		Description:      "Inserts deterministic key-value string/integer pairs into a hash table, performs lookups, and computes checksum.",
		AlgorithmPurpose: "Measures dynamic memory allocation, string hashing, bucket distribution, collision resolution, and cache locality.",
		Parameters: []WorkloadParam{
			{Name: "operations", Type: "int", Default: 300000, Description: "Number of insert and lookup operations to perform"},
		},
		SupportedModes:  []string{"standard"},
		FairnessCaveats: "Standard hash table implementations differ in hashing algorithms (e.g., SipHash in Rust/Python, AES-NI / runtime hashing in Go, SwissTable in modern C++, Murmur in Node). Differences reflect real-world runtime choices.",
		OperationsCount: func(params map[string]interface{}) int64 {
			ops := getIntParam(params, "operations", 300000)
			return int64(ops)
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			return 0
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, "Expected JSON output with checksum"
			}
			checksum, ok := outMap["checksum"]
			if !ok {
				return false, stdout, "Missing 'checksum' in JSON output"
			}
			return true, fmt.Sprintf("checksum=%v", checksum), "matching checksum"
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			ops := getIntParam(params, "operations", 300000)
			return []string{"--operations", fmt.Sprintf("%d", ops)}
		},
	})

	// 4. JSON Decoding and Processing
	r.Register(&WorkloadContract{
		ID:               "json_proc",
		Version:          "1.0.0",
		Name:             "JSON Decoding and Processing",
		Description:      "Parses a deterministic synthetic JSON document containing an array of records, filters active items, aggregates amounts, and computes checksum.",
		AlgorithmPurpose: "Measures string manipulation, deserialization overhead, object allocation, and field access across runtimes.",
		Parameters: []WorkloadParam{
			{Name: "items", Type: "int", Default: 50000, Description: "Number of records in JSON dataset"},
		},
		SupportedModes:  []string{"standard"},
		FairnessCaveats: "Standard library JSON decoders vary between reflection-based (Go), dynamic object parsing (Node, Python), and typed deserialization (Rust serde / nlohmann C++). Each implementation uses idiomatic standard or premier community parser.",
		OperationsCount: func(params map[string]interface{}) int64 {
			items := getIntParam(params, "items", 50000)
			return int64(items)
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			return 0
		},
		PrepareDataset: func(params map[string]interface{}, tempDir string) (string, error) {
			items := getIntParam(params, "items", 50000)
			dataPath := filepath.Join(tempDir, fmt.Sprintf("data_%d.json", items))
			f, err := os.Create(dataPath)
			if err != nil {
				return "", err
			}
			defer f.Close()

			f.WriteString("[\n")
			for i := 0; i < items; i++ {
				active := (i % 3) != 0
				amount := float64((i*17)%1000) + 0.5
				line := fmt.Sprintf(`  {"id": %d, "user": "user_%d", "amount": %.2f, "active": %t, "tags": ["tag_%d", "common"]}`,
					i, i%500, amount, active, i%10)
				if i < items-1 {
					line += ",\n"
				} else {
					line += "\n"
				}
				f.WriteString(line)
			}
			f.WriteString("]\n")
			return dataPath, nil
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, "Expected JSON output with active_count and total_amount"
			}
			activeCount, hasCount := outMap["active_count"]
			totalAmt, hasAmt := outMap["total_amount"]
			if !hasCount || !hasAmt {
				return false, stdout, "Missing active_count or total_amount"
			}
			return true, fmt.Sprintf("count=%v, total=%.2f", activeCount, toNumeric(totalAmt)), "valid JSON aggregation"
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			return []string{"--input", datasetPath}
		},
	})

	// 5. Matrix Operations
	r.Register(&WorkloadContract{
		ID:               "matrix",
		Version:          "1.0.0",
		Name:             "Matrix Multiplication",
		Description:      "Multiplies two NxN matrices of double-precision floating point numbers generated deterministically, returning the sum checksum.",
		AlgorithmPurpose: "Measures tight loop vectorization, CPU floating point compute units (FMA), and memory striding/cache misses.",
		Parameters: []WorkloadParam{
			{Name: "dimension", Type: "int", Default: 300, Description: "Dimension N of square NxN matrices"},
		},
		SupportedModes:  []string{"standard"},
		FairnessCaveats: "All implementations perform straightforward 3-nested-loop matrix multiplication (O(N^3)) with row-major layout without external BLAS acceleration, isolating compiler loop optimizations and instruction pipeline scheduling.",
		OperationsCount: func(params map[string]interface{}) int64 {
			n := getIntParam(params, "dimension", 300)
			return int64(2 * n * n * n) // 2N^3 FLOPs
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			return 0
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, "Expected JSON output with checksum"
			}
			checksum, ok := outMap["checksum"]
			if !ok {
				return false, stdout, "Missing checksum in matrix output"
			}
			return true, fmt.Sprintf("checksum=%v", checksum), "matching matrix checksum"
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			dim := getIntParam(params, "dimension", 300)
			return []string{"--dim", fmt.Sprintf("%d", dim)}
		},
	})

	// 6. File Processing Workload
	r.Register(&WorkloadContract{
		ID:               "file_proc",
		Version:          "1.0.0",
		Name:             "File Processing Stream",
		Description:      "Reads and streams a generated binary/text file in 64KB chunks, calculating a deterministic SHA-256 digest and byte count.",
		AlgorithmPurpose: "Measures system call efficiency, file I/O buffering, memory mapping or buffered streams, and SHA-256 cryptographic throughput.",
		Parameters: []WorkloadParam{
			{Name: "size_mb", Type: "int", Default: 25, Description: "Size in megabytes of file to generate and process"},
		},
		SupportedModes:  []string{"standard"},
		FairnessCaveats: "Operating system page caches warm file data in memory across runs. This measures user-space buffer management, system call throughput, and SHA256 pipeline, not raw physical magnetic/NAND disk latency.",
		OperationsCount: func(params map[string]interface{}) int64 {
			mb := getIntParam(params, "size_mb", 25)
			return int64(mb * 1024 * 1024)
		},
		TotalBytesCount: func(params map[string]interface{}) int64 {
			mb := getIntParam(params, "size_mb", 25)
			return int64(mb * 1024 * 1024)
		},
		PrepareDataset: func(params map[string]interface{}, tempDir string) (string, error) {
			mb := getIntParam(params, "size_mb", 25)
			filePath := filepath.Join(tempDir, fmt.Sprintf("test_file_%dmb.bin", mb))

			f, err := os.Create(filePath)
			if err != nil {
				return "", err
			}
			defer f.Close()

			// Generate deterministic data using pattern
			chunk := make([]byte, 64*1024)
			for i := range chunk {
				chunk[i] = byte((i * 31) % 256)
			}

			totalBytes := mb * 1024 * 1024
			written := 0
			for written < totalBytes {
				toWrite := len(chunk)
				if totalBytes-written < toWrite {
					toWrite = totalBytes - written
				}
				if _, err := f.Write(chunk[:toWrite]); err != nil {
					return "", err
				}
				written += toWrite
			}

			return filePath, nil
		},
		ValidateResult: func(params map[string]interface{}, stdout string) (bool, string, string) {
			var outMap map[string]interface{}
			if err := json.Unmarshal([]byte(findJSONLine(stdout)), &outMap); err != nil {
				return false, stdout, "Expected JSON with bytes_read and sha256"
			}
			bytesRead := int64(toNumeric(outMap["bytes_read"]))
			sha, _ := outMap["sha256"].(string)

			mb := getIntParam(params, "size_mb", 25)
			expectedBytes := int64(mb * 1024 * 1024)
			if bytesRead != expectedBytes {
				return false, fmt.Sprintf("bytes_read=%d", bytesRead), fmt.Sprintf("expected %d bytes", expectedBytes)
			}
			if sha == "" {
				return false, "empty sha256", "valid sha256 hex"
			}
			return true, fmt.Sprintf("bytes=%d, sha256=%s", bytesRead, sha[:8]+"..."), "correct SHA-256 digest"
		},
		ToCLIArgs: func(params map[string]interface{}, datasetPath string) []string {
			return []string{"--file", datasetPath}
		},
	})

	return r
}

func (r *Registry) Register(c *WorkloadContract) {
	if _, exists := r.workloads[c.ID]; !exists {
		r.ordered = append(r.ordered, c.ID)
	}
	r.workloads[c.ID] = c
}

func (r *Registry) Get(id string) (*WorkloadContract, error) {
	w, exists := r.workloads[id]
	if !exists {
		return nil, fmt.Errorf("workload '%s' not registered", id)
	}
	return w, nil
}

func (r *Registry) List() []*WorkloadContract {
	res := make([]*WorkloadContract, 0, len(r.ordered))
	for _, id := range r.ordered {
		res = append(res, r.workloads[id])
	}
	return res
}

// Helpers
func getIntParam(params map[string]interface{}, key string, def int) int {
	if params == nil {
		return def
	}
	val, ok := params[key]
	if !ok {
		return def
	}
	return int(toNumeric(val))
}

func getStringParam(params map[string]interface{}, key string, def string) string {
	if params == nil {
		return def
	}
	val, ok := params[key]
	if !ok {
		return def
	}
	if s, ok := val.(string); ok && s != "" {
		return s
	}
	return def
}

func toNumeric(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	default:
		return 0
	}
}

func findJSONLine(stdout string) string {
	for _, line := range splitLines(stdout) {
		trimmed := trimSpaces(line)
		if len(trimmed) > 0 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
			return trimmed
		}
	}
	return stdout
}

func splitLines(s string) []string {
	var lines []string
	curr := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, curr)
			curr = ""
		} else {
			curr += string(s[i])
		}
	}
	if curr != "" {
		lines = append(lines, curr)
	}
	return lines
}

func trimSpaces(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// ComputeFileSHA256 returns hex encoded SHA256 of a file.
func ComputeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
