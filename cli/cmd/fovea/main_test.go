package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

const validComplete = "../../internal/core/testdata/valid_complete"

func exit(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// Asking for help is not misuse: both help forms exit 0; misuse exits 2.
func TestHelpExitsZeroMisuseExitsTwo(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--help-issues"}} {
		if code, out, _ := exit(args...); code != 0 || out == "" {
			t.Errorf("%v: exit %d (stdout %d bytes), want 0 with usage on stdout", args, code, len(out))
		}
	}
	for _, args := range [][]string{{}, {"bogus"}} {
		if code, _, _ := exit(args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

// A flag a command does not take is refused, not silently ignored.
func TestFlagsAreCheckedPerCommand(t *testing.T) {
	for _, args := range [][]string{
		{"lint", "--json", validComplete},
		{"lint", "--html", validComplete},
		{"score", "--github", validComplete},
		{"score", "--html", validComplete},
		{"render", "--github", validComplete},
		{"init", "--json", filepath.Join(t.TempDir(), "x")},
	} {
		code, _, errs := exit(args...)
		if code != 2 || !strings.Contains(errs, "does not take") {
			t.Errorf("%v: exit %d, stderr %q; want 2 naming the refused flag", args, code, errs)
		}
	}
}

func TestAcceptedFlagsStillWork(t *testing.T) {
	for _, args := range [][]string{
		{"lint", "--github", validComplete},
		{"score", "--json", validComplete},
		{"render", "--format", "md", validComplete},
		{"render", "--format=json", validComplete},
		{"render", "--html", validComplete},
	} {
		if code, _, errs := exit(args...); code != 0 {
			t.Errorf("%v: exit %d: %s", args, code, errs)
		}
	}
}
