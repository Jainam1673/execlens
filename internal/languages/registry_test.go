package languages

import (
	"context"
	"testing"
)

func TestLanguageRegistry(t *testing.T) {
	reg := NewDefaultRegistry()
	list := reg.List()
	if len(list) != 6 {
		t.Fatalf("expected 6 default language adapters, got %d", len(list))
	}

	for _, expectedID := range []string{"cpp", "rust", "go", "java", "python", "javascript"} {
		adapter, err := reg.Get(expectedID)
		if err != nil {
			t.Fatalf("failed to find adapter for %s: %v", expectedID, err)
		}
		if adapter.ID() != expectedID {
			t.Errorf("adapter ID mismatch: %s != %s", adapter.ID(), expectedID)
		}
	}

	// Test detection
	ctx := context.Background()
	results := reg.DetectAll(ctx)
	if len(results) != 6 {
		t.Fatalf("expected 6 detection results, got %d", len(results))
	}

	for _, info := range results {
		t.Logf("Language: %s, Available: %v, Version: %s, Missing: %s", info.DisplayName, info.IsAvailable, info.Version, info.MissingReason)
		// On this machine, all 6 should be available!
		if !info.IsAvailable {
			t.Errorf("expected %s to be available, but missing reason: %s", info.ID, info.MissingReason)
		}
	}
}
