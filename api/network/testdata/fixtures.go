// Package testdata provides test fixtures for Network API tests.
// All JSON files contain real API responses captured from UniFi controllers.
package testdata

import (
	"embed"
	"encoding/json"
	"path/filepath"
	"testing"
)

// FS embeds all JSON fixture files.
//
//go:embed **/*.json
var FS embed.FS

// LoadFixture reads and returns fixture content as string.
// The path should be relative to testdata directory (e.g., "sites/list_success.json").
func LoadFixture(t *testing.T, path string) string {
	t.Helper()

	data, err := FS.ReadFile(filepath.Join(path))
	if err != nil {
		t.Fatalf("failed to load fixture %s: %v", path, err)
	}

	return string(data)
}

// LoadFixtureJSON reads fixture and unmarshals into provided value.
// Useful for testing deserialization or when you need structured data.
func LoadFixtureJSON(t *testing.T, path string, v interface{}) {
	t.Helper()

	data := LoadFixture(t, path)
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
}
