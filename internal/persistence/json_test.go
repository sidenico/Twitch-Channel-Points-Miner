package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	data := map[string]interface{}{"foo": "bar", "num": float64(1)}

	if err := SaveJSON(path, data); err != nil {
		t.Fatalf("save json error: %v", err)
	}
	var decoded map[string]interface{}
	if err := LoadJSON(path, &decoded); err != nil {
		t.Fatalf("load json error: %v", err)
	}
	if decoded["foo"] != "bar" || decoded["num"] != float64(1) {
		t.Fatalf("decoded content mismatch: %#v", decoded)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}
