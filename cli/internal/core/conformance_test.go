package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// A case under testdata/<name>/ is an assessment directory described by
// case.yaml. When base is set, the case is the base case's files overlaid
// with the case's own files, minus the paths listed in remove.
// expect.errors is the exact multiset of (rule, where) pairs lint must raise
// as errors: no more, no fewer, each as often as listed. where is
// "fovea.yaml" for header findings, the cell file name for a file that
// cannot be tied to an id, and the cell id otherwise. The format is plain
// YAML on purpose; these cases seed the conformance suite every fovea
// implementation must pass, and expect: leaves room for score and states.
type caseSpec struct {
	Describes string   `yaml:"describes"`
	Base      string   `yaml:"base"`
	Remove    []string `yaml:"remove"`
	Expect    struct {
		Errors []expectedFinding `yaml:"errors"`
	} `yaml:"expect"`
}

type expectedFinding struct {
	Rule  string `yaml:"rule"`
	Where string `yaml:"where"`
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

// errorPairs is the sorted multiset of "rule @ where" for error findings.
func errorPairs(fs []Finding) []string {
	out := []string{}
	for _, f := range fs {
		if f.Err {
			out = append(out, f.Rule+" @ "+f.Where)
		}
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
// case's expected (rule, where) multiset.
func TestConformanceCases(t *testing.T) {
	for _, name := range caseNames(t) {
		t.Run(name, func(t *testing.T) {
			spec := readCase(t, name)
			dir := materialize(t, name)
			_, _, fs, err := Check(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			want := []string{}
			for _, e := range spec.Expect.Errors {
				want = append(want, e.Rule+" @ "+e.Where)
			}
			sort.Strings(want)
			if got := errorPairs(fs); !reflect.DeepEqual(got, want) {
				t.Errorf("%s\nerrors: got %v, want %v", spec.Describes, got, want)
				for _, f := range fs {
					t.Logf("  %s %s: %s", f.Rule, f.Where, f.Message)
				}
			}
		})
	}
}

// TestEveryRuleHasACaseNamedAfterIt keeps the documented promise: each rule
// the lint can raise has testdata/<code>/, and that case expects it. It
// iterates the rule table, which errf requires, so no rule can be missed.
func TestEveryRuleHasACaseNamedAfterIt(t *testing.T) {
	if len(allRules) == 0 {
		t.Fatal("the rule table is empty")
	}
	seen := map[string]bool{}
	for _, r := range allRules {
		code := r.Code()
		if seen[code] {
			t.Errorf("rule code %s is in the table twice", code)
		}
		seen[code] = true
		if _, err := os.Stat(filepath.Join("testdata", code, "case.yaml")); err != nil {
			t.Errorf("rule %s has no case testdata/%s/", code, code)
			continue
		}
		expects := false
		for _, e := range readCase(t, code).Expect.Errors {
			expects = expects || e.Rule == code
		}
		if !expects {
			t.Errorf("case testdata/%s/ does not expect rule %s", code, code)
		}
	}
}

var ruleForgery = regexp.MustCompile(`\bnewRule\(|\bRule\{`)

// TestRulesAreMadeOnlyInTheTable closes the gap a table could leave: a Rule
// built anywhere but rules.go would reach errf without being in allRules.
func TestRulesAreMadeOnlyInTheTable(t *testing.T) {
	srcs, _ := filepath.Glob("*.go")
	for _, p := range srcs {
		if p == "rules.go" || strings.HasSuffix(p, "_test.go") {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if ruleForgery.MatchString(line) {
				t.Errorf("%s:%d makes a Rule outside rules.go: %s", p, i+1, strings.TrimSpace(line))
			}
		}
	}
}
