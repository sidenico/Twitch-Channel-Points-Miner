package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	data := map[string]interface{}{"foo": "bar", "num": 1}

	if err := SaveJSON(path, data); err != nil {
		t.Fatalf("save json error: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back error: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded["foo"] != "bar" || decoded["num"] != float64(1) {
		t.Fatalf("decoded content mismatch: %#v", decoded)
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "in.json")
	if err := os.WriteFile(path, []byte(`{"hello":"world","n":2}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	var decoded map[string]interface{}
	if err := LoadJSON(path, &decoded); err != nil {
		t.Fatalf("load json error: %v", err)
	}
	if decoded["hello"] != "world" || decoded["n"] != float64(2) {
		t.Fatalf("decoded content mismatch: %#v", decoded)
	}
}
