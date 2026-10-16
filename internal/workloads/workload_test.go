package workloads

import (
	"path/filepath"
	"testing"
)

func TestWorkloadsRegistry(t *testing.T) {
	reg := NewRegistry()
	list := reg.List()
	if len(list) != 6 {
		t.Fatalf("expected 6 workloads, got %d", len(list))
	}

	expectedIDs := []string{"sieve", "sort", "hashmap", "json_proc", "matrix", "file_proc"}
	for _, id := range expectedIDs {
		w, err := reg.Get(id)
		if err != nil {
			t.Fatalf("expected workload %s to exist: %v", id, err)
		}
		if w.ID != id {
			t.Errorf("expected ID %s, got %s", id, w.ID)
		}
	}
}

func TestSieveValidation(t *testing.T) {
	reg := NewRegistry()
	sieve, _ := reg.Get("sieve")

	params := map[string]interface{}{"limit": 100}
	// Primes up to 100: 25 primes (2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97)
	validJSON := `{"workload":"sieve","limit":100,"count":25}`
	passed, actual, expected := sieve.ValidateResult(params, validJSON)
	if !passed {
		t.Errorf("validation should pass: actual=%s, expected=%s", actual, expected)
	}

	invalidJSON := `{"workload":"sieve","limit":100,"count":24}`
	passed2, _, _ := sieve.ValidateResult(params, invalidJSON)
	if passed2 {
		t.Errorf("validation should fail for incorrect prime count")
	}
}

func TestFileProcDatasetPreparation(t *testing.T) {
	reg := NewRegistry()
	fileProc, _ := reg.Get("file_proc")

	tmp := t.TempDir()
	params := map[string]interface{}{"size_mb": 1}
	path, err := fileProc.PrepareDataset(params, tmp)
	if err != nil {
		t.Fatalf("failed to prepare dataset: %v", err)
	}
	if filepath.Dir(path) != tmp {
		t.Errorf("expected file in temp dir %s, got %s", tmp, path)
	}
}
