package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

// A case under testdata/<name>/ is an assessment directory described by
// case.yaml. When base is set, the case is the base case's files overlaid
// with the case's own files, minus the paths listed in remove. errors is the
// exact set of rule codes lint must raise as errors: no more, no fewer.
// The format is language-neutral on purpose; these cases seed the
// conformance suite every fovea implementation must pass.
type caseSpec struct {
	Describes string   `yaml:"describes"`
	Base      string   `yaml:"base"`
	Remove    []string `yaml:"remove"`
	Errors    []string `yaml:"errors"`
}

func readCase(t *testing.T, name string) caseSpec {
	t.Helper()
	var spec caseSpec
	b, err := os.ReadFile(filepath.Join("testdata", name, "case.yaml"))
	if err != nil {
		t.Fatalf("case %s: %v", name, err)
	}
	if err := yaml.Unmarshal(b, &spec); err != nil {
		t.Fatalf("case %s: case.yaml: %v", name, err)
	}
	return spec
}

// materialize builds the case's assessment directory in a fresh temp dir,
// so tests that write (init) never touch the committed fixtures.
func materialize(t *testing.T, name string) string {
	t.Helper()
	spec := readCase(t, name)
	dst := t.TempDir()
	if spec.Base != "" {
		copyTree(t, filepath.Join("testdata", spec.Base), dst)
	}
	copyTree(t, filepath.Join("testdata", name), dst)
	for _, p := range spec.Remove {
		if err := os.Remove(filepath.Join(dst, p)); err != nil {
			t.Fatalf("case %s: remove %s: %v", name, p, err)
		}
	}
	return dst
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "case.yaml" {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
}

func errorRules(fs []Finding) []string {
	set := map[string]bool{}
	for _, f := range fs {
		if f.Err {
			set[f.Rule] = true
		}
	}
	out := []string{}
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func caseNames(t *testing.T) []string {
	t.Helper()
	ents, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		if _, err := os.Stat(filepath.Join("testdata", e.Name(), "case.yaml")); e.IsDir() && err == nil {
			names = append(names, e.Name())
		}
	}
	return names
}

// TestConformanceCases runs lint over every case and requires exactly the
// case's expected error rules.
func TestConformanceCases(t *testing.T) {
	for _, name := range caseNames(t) {
		t.Run(name, func(t *testing.T) {
			spec := readCase(t, name)
			dir := materialize(t, name)
			_, _, fs, err := Check(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			want := append([]string{}, spec.Errors...)
			sort.Strings(want)
			if got := errorRules(fs); !reflect.DeepEqual(got, want) {
				t.Errorf("%s\nerror rules: got %v, want %v", spec.Describes, got, want)
				for _, f := range fs {
					t.Logf("  %s %s: %s", f.Rule, f.Where, f.Message)
				}
			}
		})
	}
}
