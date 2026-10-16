package reports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// VerifyReport performs a comprehensive audit on generated report artifacts and README integration.
func VerifyReport(reportDir string, readmePath string) error {
	// 1. Verify Core Files
	coreFiles := []string{
		"summary.json",
		"raw-results.json",
		"comparison.csv",
		"manifest.json",
		"report.md",
		"report.html",
		"charts/overview.svg",
		"charts/runtime-comparison.svg",
		"charts/throughput-comparison.svg",
		"charts/memory-comparison.svg",
		"charts/variability.svg",
		"charts/scaling.svg",
		"charts/startup-comparison.svg",
		"charts/build-comparison.svg",
		"charts/workload-heatmap.svg",
	}

	for _, rel := range coreFiles {
		p := filepath.Join(reportDir, rel)
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("missing required report artifact: %s (%w)", rel, err)
		}
		if info.Size() == 0 {
			return fmt.Errorf("artifact is empty (0 bytes): %s", rel)
		}
	}

	// 2. Validate JSON Schemas and Values (no NaN/Inf)
	if err := validateJSONNoNaN(filepath.Join(reportDir, "summary.json")); err != nil {
		return fmt.Errorf("summary.json validation failed: %w", err)
	}
	if err := validateJSONNoNaN(filepath.Join(reportDir, "raw-results.json")); err != nil {
		return fmt.Errorf("raw-results.json validation failed: %w", err)
	}
	if err := validateJSONNoNaN(filepath.Join(reportDir, "manifest.json")); err != nil {
		return fmt.Errorf("manifest.json validation failed: %w", err)
	}

	// 3. Validate Manifest Artifact Hashes
	manifestPath := filepath.Join(reportDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ReportManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	for _, art := range manifest.Artifacts {
		artPath := filepath.Join(reportDir, art.Path)
		data, err := os.ReadFile(artPath)
		if err != nil {
			return fmt.Errorf("manifest references unreadable artifact: %s (%w)", art.Path, err)
		}
		computedHash := sha256.Sum256(data)
		computedHex := hex.EncodeToString(computedHash[:])
		if computedHex != art.SHA256 {
			return fmt.Errorf("checksum mismatch for artifact %s: recorded %s, computed %s", art.Path, art.SHA256, computedHex)
		}
	}

	// 4. Validate Markdown Image & File Links in report.md
	mdPath := filepath.Join(reportDir, "report.md")
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read report.md: %w", err)
	}

	imgRe := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	matches := imgRe.FindAllStringSubmatch(string(mdContent), -1)
	for _, m := range matches {
		ref := m[1]
		if !strings.HasPrefix(ref, "http://") && !strings.HasPrefix(ref, "https://") {
			target := filepath.Join(reportDir, ref)
			if _, err := os.Stat(target); err != nil {
				return fmt.Errorf("report.md contains broken image link: %s -> %s", ref, target)
			}
		}
	}

	// 5. Validate README.md Markers & Embedded Image Links
	if readmePath != "" {
		readmeData, err := os.ReadFile(readmePath)
		if err != nil {
			return fmt.Errorf("failed to read README file at %s: %w", readmePath, err)
		}

		content := string(readmeData)
		startMarker := "<!-- EXECLENS_REPORT_START -->"
		endMarker := "<!-- EXECLENS_REPORT_END -->"

		startIdx := strings.Index(content, startMarker)
		endIdx := strings.Index(content, endMarker)

		if startIdx == -1 {
			return fmt.Errorf("README.md is missing %s marker", startMarker)
		}
		if endIdx == -1 {
			return fmt.Errorf("README.md is missing %s marker", endMarker)
		}
		if endIdx <= startIdx {
			return fmt.Errorf("README.md markers are inverted or malformed (start: %d, end: %d)", startIdx, endIdx)
		}

		// Check README image references inside the generated section
		section := content[startIdx:endIdx]
		readmeMatches := imgRe.FindAllStringSubmatch(section, -1)
		for _, m := range readmeMatches {
			ref := m[1]
			if !strings.HasPrefix(ref, "http://") && !strings.HasPrefix(ref, "https://") {
				// Base directory of README.md is repo root
				target := filepath.Join(filepath.Dir(readmePath), ref)
				if _, err := os.Stat(target); err != nil {
					return fmt.Errorf("README.md contains broken image reference: %s -> %s", ref, target)
				}
			}
		}
	}

	return nil
}

func validateJSONNoNaN(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Check for literal NaN or Infinity strings
	str := string(data)
	if strings.Contains(str, "NaN") || strings.Contains(str, "Infinity") {
		return fmt.Errorf("file %s contains invalid non-finite values (NaN/Infinity)", path)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("file %s is invalid JSON: %w", path, err)
	}

	return checkNoNaNRecursive(v)
}

func checkNoNaNRecursive(v interface{}) error {
	switch val := v.(type) {
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return fmt.Errorf("encountered non-finite float: %v", val)
		}
	case []interface{}:
		for _, elem := range val {
			if err := checkNoNaNRecursive(elem); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for _, elem := range val {
			if err := checkNoNaNRecursive(elem); err != nil {
				return err
			}
		}
	}
	return nil
}
