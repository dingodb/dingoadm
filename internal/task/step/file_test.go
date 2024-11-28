package step

import (
	"fmt"
	"regexp"
	"testing"
)

/*
* TestKvSplit, run: go test ./internal/task/step -run ^TestKvSplit$
 */
func TestKvSplit(t *testing.T) {
	REGEX_KV_SPLIT := `^(\s*[^%s]+)%s\s*([^#\s]*)`
	line := "  mdsAddr: ${cluster_mds_addr}"

	// Compile the regex
	pattern := fmt.Sprintf(REGEX_KV_SPLIT, ": ", ": ")
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}

	// Find matches
	matches := regex.FindStringSubmatch(line)
	if len(matches) != 3 { // Total match + 2 groups
		t.Fatalf("Expected 3 matches, got %d", len(matches))
	}

	// Preserve key format with leading spaces
	rawKey := matches[1] // Includes leading spaces
	value := matches[2]

	// Assertions
	if rawKey != "  mdsAddr" { // Notice the two spaces
		t.Errorf("Expected key '  mdsAddr', got '%s'", rawKey)
	}
	if value != "${cluster_mds_addr}" {
		t.Errorf("Expected value '${cluster_mds_addr}', got '%s'", value)
	}

	fmt.Printf("Key: '%s', Value: '%s'\n", rawKey, value)

}
