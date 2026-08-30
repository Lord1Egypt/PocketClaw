package pcruntime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// newTestManifest builds a catalog over fake system tools so the execution
// mechanics can be tested without depending on which binaries a build host has.
func newTestManifest(t *testing.T, tools ...Tool) *Manifest {
	t.Helper()
	manifest := &Manifest{
		RuntimeVersion: ManifestVersion,
		CatalogVersion: "test",
		Tools:          tools,
	}
	if err := manifest.validate(); err != nil {
		t.Fatalf("test manifest is invalid: %v", err)
	}
	return manifest
}

func systemTool(id string, profile TimeoutProfile, maxOutput int64) Tool {
	return Tool{
		ToolID:         id,
		DisplayName:    id,
		CommandName:    id,
		Version:        "platform",
		ABI:            "arm64-v8a",
		Delivery:       DeliverySystem,
		TrustedSource:  "test",
		Capabilities:   []string{"test.execute"},
		TimeoutProfile: profile,
		MaxOutputBytes: maxOutput,
		SecurityClass:  SecurityClassSystem,
	}
}

// useFakeSystemBin points system resolution at dir for the duration of the test.
func useFakeSystemBin(t *testing.T, dir string) {
	t.Helper()
	original := systemBinDirs
	systemBinDirs = []string{dir}
	t.Cleanup(func() { systemBinDirs = original })
}

// writeScript installs an executable shell script into a fake system bin dir.
func writeScript(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatalf("cannot write test script %s: %v", name, err)
	}
}

// newTestManager wires a manager over a fake system bin dir and a workspace.
func newTestManager(t *testing.T, manifest *Manifest) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	workspace := filepath.Join(root, "workspace")
	metadata := filepath.Join(root, "runtime")
	for _, dir := range []string{binDir, workspace, metadata} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("cannot create %s: %v", dir, err)
		}
	}
	useFakeSystemBin(t, binDir)

	registry, err := NewRegistryWith(manifest, &Paths{MetadataDir: metadata, Workspace: workspace})
	if err != nil {
		t.Fatalf("cannot build registry: %v", err)
	}
	manager, err := NewManagerWith(registry, workspace)
	if err != nil {
		t.Fatalf("cannot build manager: %v", err)
	}
	return manager, binDir
}

// decodeJSONLines parses a logger file sink into structured records.
func decodeJSONLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read log sink: %v", err)
	}
	var records []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	for decoder.More() {
		var record map[string]any
		if err := decoder.Decode(&record); err != nil {
			t.Fatalf("log sink is not JSON lines: %v", err)
		}
		records = append(records, record)
	}
	return records
}
