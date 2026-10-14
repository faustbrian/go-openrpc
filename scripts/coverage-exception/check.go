//go:build ignore

// Package main verifies fresh complete native coverage at the owner boundary.
// It is not public or a replacement for the shared collector's other gates.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const recordPath = "verification/coverage-exception.json"
const modulePath = "github.com/faustbrian/go-openrpc/v2"

var allowed = map[string]bool{
	modulePath + "/serialization.go:47.3,48.1": true,
	modulePath + "/serialization.go:53.3,54.1": true,
}

// Record binds only material executable inputs and the native compiler geometry.
type Record struct {
	// Inputs binds material source, dependency and executable policy bytes.
	Inputs string `json:"inputs_sha256"`
	// Geometry binds every native compiler target and statement count.
	Geometry string `json:"geometry_sha256"`
	// Expires is the exclusive UTC date boundary for this narrow proof.
	Expires string `json:"expires"`
	// Proof records the independently reviewed codec domain and limitations.
	Proof string `json:"proof"`
}

type packageInfo struct {
	ImportPath, Dir   string
	GoFiles, CgoFiles []string
	Error             *struct{ Err string }
}
type packagePolicy struct {
	ImportPath string `json:"import_path"`
	Required   bool   `json:"coverage_required"`
}
type block struct {
	Statements uint64
	Count      uint64
}

// command reads native metadata without introducing a replacement or workspace.
// #nosec G204 -- fixed Go metadata arguments at internal call sites, never profile or record content
func command(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return exec.CommandContext(ctx, "go", args...).Output()
}

// inventory reads the selected production packages and their actual source files.
func inventory() ([]string, map[string]bool, []string, error) {
	data, err := os.ReadFile("packages.json")
	if err != nil {
		return nil, nil, nil, err
	}
	var catalog struct {
		Packages []packagePolicy `json:"packages"`
	}
	if err = json.Unmarshal(data, &catalog); err != nil {
		return nil, nil, nil, err
	}
	targets := []string{}
	want := map[string]bool{}
	for _, p := range catalog.Packages {
		if p.Required {
			if want[p.ImportPath] || !strings.HasPrefix(p.ImportPath, modulePath) {
				return nil, nil, nil, errors.New("invalid coverage inventory")
			}
			want[p.ImportPath] = true
			targets = append(targets, p.ImportPath)
		}
	}
	sort.Strings(targets)
	if len(targets) == 0 {
		return nil, nil, nil, errors.New("empty coverage inventory")
	}
	data, err = command("list", "-json", "./...")
	if err != nil {
		return nil, nil, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	files := map[string]bool{}
	inputs := []string{}
	for {
		var p packageInfo
		err = decoder.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, nil, err
		}
		if p.Error != nil || !strings.HasPrefix(p.ImportPath, modulePath) {
			return nil, nil, nil, errors.New("invalid selected package")
		}
		for _, name := range append(p.GoFiles, p.CgoFiles...) {
			inputs = append(inputs, filepath.Join(p.Dir, name))
			if want[p.ImportPath] {
				files[p.ImportPath+"/"+name] = true
			}
		}
		delete(want, p.ImportPath)
	}
	if len(want) != 0 || len(files) == 0 {
		return nil, nil, nil, errors.New("missing production package")
	}
	return targets, files, inputs, nil
}

// inputDigest automatically expires the proof for changed source, dependencies,
// checker/test policy, native compiler version or active standard-library codec.
func inputDigest(inputs []string) (string, error) {
	if runtime.Version() != "go1.27.2" {
		return "", errors.New("requires Go 1.27.2")
	}
	flags, err := command("env", "GOFLAGS")
	if err != nil {
		return "", err
	}
	for _, f := range strings.Fields(string(flags)) {
		if f != "-mod=readonly" && f != "-mod=mod" && !regexp.MustCompile(`^-p=[1-9][0-9]*$`).MatchString(f) {
			return "", errors.New("custom build flags expire the proof")
		}
	}
	data, err := command("env", "GOEXPERIMENT", "GOWORK")
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 || lines[0] != "off" {
		return "", errors.New("requires default experiments and GOWORK=off")
	}
	data, err = command("list", "-deps", "-json", "encoding/json", "encoding/json/v2", "encoding/json/jsontext", "bytes")
	if err != nil {
		return "", err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	stdlib := map[string]string{}
	active := false
	for {
		var p packageInfo
		err = d.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if p.Error != nil {
			return "", errors.New("invalid codec selection")
		}
		if p.ImportPath != "bytes" && p.ImportPath != "encoding/json" && !strings.HasPrefix(p.ImportPath, "encoding/json/") {
			continue
		}
		for _, name := range p.GoFiles {
			if p.ImportPath == "encoding/json" && name == "v2_stream.go" {
				active = true
			}
			stdlib[filepath.Join(p.Dir, name)] = p.ImportPath + "/" + name
		}
	}
	if !active {
		return "", errors.New("default JSONv2 codec required")
	}
	inputs = append(inputs, "go.mod", "go.sum", "packages.json", ".golib.yaml", "verification/package.mk", "scripts/check-coverage.sh", "scripts/coverage-exception/check.go", "scripts/coverage-exception/check_test.go", "parse/required_constructor_boundary_test.go", "serialization_depth_boundary_test.go")
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}
	entries := map[string]string{}
	for _, path := range inputs {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		name, err := filepath.Rel(root, absolute)
		if err != nil || strings.HasPrefix(name, "../") {
			return "", errors.New("source outside owner")
		}
		entries[absolute] = filepath.ToSlash(name)
	}
	for path, name := range stdlib {
		entries[path] = "stdlib/" + name
	}
	names := make([]string, 0, len(entries))
	digests := map[string]string{}
	for path, name := range entries {
		// #nosec G304 -- actual owner and standard-library source paths from native Go metadata, not an input sandbox
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		names = append(names, name)
		digests[name] = hex.EncodeToString(sum[:])
	}
	sort.Strings(names)
	h := sha256.New()
	_, _ = fmt.Fprintln(h, runtime.Version())
	for _, name := range names {
		_, _ = fmt.Fprintf(h, "%s  %s\n", digests[name], name)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var location = regexp.MustCompile(`^(.+\.go):([1-9][0-9]*)\.([1-9][0-9]*),([1-9][0-9]*)\.([1-9][0-9]*)$`)

// profileBlocks merges legitimate identical cross-test-binary blocks, as native
// Go coverage does. Conflicting statement counts and foreign files are fatal.
func profileBlocks(reader io.Reader, files map[string]bool) (map[string]block, error) {
	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() || scanner.Text() != "mode: atomic" {
		return nil, errors.New("atomic compiler profile required")
	}
	blocks := map[string]block{}
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			return nil, errors.New("malformed profile")
		}
		match := location.FindStringSubmatch(parts[0])
		if match == nil || !files[match[1]] {
			return nil, errors.New("foreign/malformed profile target")
		}
		first, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, err
		}
		last, err := strconv.Atoi(match[4])
		if err != nil || last < first {
			return nil, errors.New("invalid profile range")
		}
		n, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, errors.New("invalid statement count")
		}
		c, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return nil, errors.New("invalid execution count")
		}
		old, exists := blocks[parts[0]]
		if exists && (old.Statements != n || ^uint64(0)-old.Count < c) {
			return nil, errors.New("ambiguous/overflow profile block")
		}
		blocks[parts[0]] = block{n, old.Count + c}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, errors.New("empty or missing production file")
	}
	return blocks, nil
}

// geometryDigest binds every actual compiler block independently of hit counts.
func geometryDigest(blocks map[string]block) string {
	keys := make([]string, 0, len(blocks))
	for key := range blocks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, key := range keys {
		_, _ = fmt.Fprintf(h, "%s %d\n", key, blocks[key].Statements)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// verify rejects every gap except the two reviewed single-statement codec arms.
func verify(blocks map[string]block, geometry string) error {
	if geometryDigest(blocks) != geometry {
		return errors.New("compiler geometry changed/missing block")
	}
	for key := range allowed {
		b, ok := blocks[key]
		if !ok || b.Statements != 1 {
			return errors.New("reviewed block missing/changed")
		}
	}
	for key, b := range blocks {
		if b.Statements > 0 && b.Count == 0 && !allowed[key] {
			return fmt.Errorf("uncovered production block: %s", key)
		}
	}
	return nil
}

func main() {
	targetsOnly := flag.Bool("targets", false, "print native production coverage targets")
	profile := flag.String("profile", "", "fresh complete atomic Go coverage profile")
	flag.Parse()
	err := run(*targetsOnly, *profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverage refusal:", err)
		os.Exit(1)
	}
}

// run admits the exact bound owner inputs before either selecting or checking.
func run(targetsOnly bool, path string) error {
	data, err := os.ReadFile(recordPath)
	if err != nil {
		return err
	}
	record, err := decodeRecord(data)
	if err != nil {
		return err
	}
	expires, err := time.Parse("2006-01-02", record.Expires)
	if err != nil || !time.Now().Before(expires) {
		return errors.New("coverage proof expired")
	}
	targets, files, inputs, err := inventory()
	if err != nil {
		return err
	}
	digest, err := inputDigest(inputs)
	if err != nil {
		return err
	}
	if record.Inputs != digest || record.Proof == "" {
		return errors.New("material input binding changed")
	}
	if targetsOnly {
		fmt.Println(strings.Join(targets, ","))
		return nil
	}
	// #nosec G304 -- explicit owner-collected profile path; this verifier is not a filesystem sandbox
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 128<<20 {
		return errors.New("invalid profile file")
	}
	blocks, err := profileBlocks(file, files)
	if err != nil {
		return err
	}
	if err = verify(blocks, record.Geometry); err != nil {
		return err
	}
	fmt.Printf("Exact owner coverage: %d compiler blocks; only reviewed codec error arms may be unexecuted\n", len(blocks))
	return nil
}

// decodeRecord rejects duplicate, unknown, missing and non-string record fields.
func decodeRecord(data []byte) (Record, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return Record{}, errors.New("invalid exception record")
	}
	values := map[string]string{}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return Record{}, err
		}
		key, ok := token.(string)
		if !ok {
			return Record{}, errors.New("invalid record key")
		}
		if _, exists := values[key]; exists {
			return Record{}, errors.New("duplicate record field")
		}
		switch key {
		case "inputs_sha256", "geometry_sha256", "expires", "proof":
		default:
			return Record{}, errors.New("unknown record field")
		}
		var value string
		if err = d.Decode(&value); err != nil {
			return Record{}, err
		}
		values[key] = value
	}
	if _, err = d.Token(); err != nil {
		return Record{}, err
	}
	var extra any
	if d.Decode(&extra) != io.EOF || len(values) != 4 {
		return Record{}, errors.New("incomplete/trailing record")
	}
	return Record{values["inputs_sha256"], values["geometry_sha256"], values["expires"], values["proof"]}, nil
}
