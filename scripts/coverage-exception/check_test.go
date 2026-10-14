//go:build ignore

package main

import (
	"strings"
	"testing"
)

// TestStrictProfileRefusesAmbiguousOrIncompleteEvidence protects owner outcomes,
// including legitimate native cross-test-binary count aggregation.
func TestStrictProfileRefusesAmbiguousOrIncompleteEvidence(t *testing.T) {
	file := modulePath + "/serialization.go"
	control := "mode: atomic\n" + file + ":47.3,48.1 1 0\n" + file + ":53.3,54.1 1 0\n" + file + ":55.2,55.65 1 2\n"
	files := map[string]bool{file: true}
	blocks, err := profileBlocks(strings.NewReader(control), files)
	if err != nil {
		t.Fatal(err)
	}
	geometry := geometryDigest(blocks)
	if err = verify(blocks, geometry); err != nil {
		t.Fatal(err)
	}
	duplicate := control + file + ":55.2,55.65 1 3\n"
	combined, err := profileBlocks(strings.NewReader(duplicate), files)
	if err != nil || combined[file+":55.2,55.65"].Count != 5 || verify(combined, geometry) != nil {
		t.Fatalf("native repeated block aggregation failed: %v", err)
	}
	for name, input := range map[string]string{
		"empty": "", "malformed": "mode: atomic\nnot a block\n", "mode": strings.Replace(control, "atomic", "set", 1),
		"foreign":             control + modulePath + "/foreign.go:1.1,2.1 1 1\n",
		"statements conflict": control + file + ":55.2,55.65 2 1\n",
		"negative count":      strings.Replace(control, "1 2", "1 -2", 1),
		"overflow":            strings.Replace(control, "1 2", "1 18446744073709551615", 1) + file + ":55.2,55.65 1 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := profileBlocks(strings.NewReader(input), files); err == nil {
				t.Fatal("malformed profile accepted")
			}
		})
	}
	for name, input := range map[string]string{
		"missing block":             strings.Replace(control, file+":55.2,55.65 1 2\n", "", 1),
		"extra uncovered":           strings.Replace(control, "1 2", "1 0", 1),
		"changed geometry":          strings.Replace(control, ":55.2,55.65", ":56.2,56.65", 1),
		"wrong reviewed statements": strings.Replace(control, ":47.3,48.1 1 0", ":47.3,48.1 2 0", 1),
	} {
		t.Run(name, func(t *testing.T) {
			b, err := profileBlocks(strings.NewReader(input), files)
			if err == nil {
				err = verify(b, geometry)
			}
			if err == nil {
				t.Fatal("unsupported production gap/geometry accepted")
			}
		})
	}
}

// TestRecordRefusesAmbiguousDispositions keeps the narrow proof immutable at use.
func TestRecordRefusesAmbiguousDispositions(t *testing.T) {
	control := `{"inputs_sha256":"input","geometry_sha256":"geometry","expires":"2026-11-14","proof":"two codec arms only"}`
	if _, err := decodeRecord([]byte(control)); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{`{}`, control + `{}`, strings.Replace(control, `"proof":"two codec arms only"`, `"proof":"a","proof":"b"`, 1), strings.Replace(control, `"proof"`, `"unscoped"`, 1)} {
		if _, err := decodeRecord([]byte(input)); err == nil {
			t.Fatal("ambiguous/unsupported disposition accepted")
		}
	}
}

// FuzzProfileAndRecordNeverPanic exercises bounded hostile owner evidence. It
// does not certify a profile's completeness or replace strict outcome controls.
func FuzzProfileAndRecordNeverPanic(f *testing.F) {
	f.Add([]byte("mode: atomic\n" + modulePath + "/serialization.go:47.3,48.1 1 0\n"))
	f.Add([]byte(`{"proof":"a","proof":"b"}`))
	f.Add([]byte("mode: atomic\nmalformed\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 32<<10 {
			return
		}
		_, _ = profileBlocks(strings.NewReader(string(data)), map[string]bool{modulePath + "/serialization.go": true})
		_, _ = decodeRecord(data)
	})
}
