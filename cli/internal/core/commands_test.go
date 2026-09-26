package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScoreJSONIsValidWhenLintFails(t *testing.T) {
	dir := materialize(t, "roadmap_without_review_by")
	var out, errb bytes.Buffer
	if code := Score(dir, true, &out, &errb); code != 1 {
		t.Fatalf("exit %d, want 1 (lint fails)", code)
	}
	if !json.Valid(out.Bytes()) {
		t.Fatalf("stdout is not valid JSON:\n%s", out.String())
	}
	if !strings.Contains(errb.String(), "lint error") {
		t.Fatalf("lint note missing from stderr: %q", errb.String())
	}
}

func TestLoadErrorsGoToStderr(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nowhere")
	for name, run := range map[string]func(o, e *bytes.Buffer) int{
		"lint":   func(o, e *bytes.Buffer) int { return Lint(missing, false, o, e) },
		"score":  func(o, e *bytes.Buffer) int { return Score(missing, true, o, e) },
		"render": func(o, e *bytes.Buffer) int { return Render(missing, false, false, o, e) },
		"init":   func(o, e *bytes.Buffer) int { return Init(missing, o, e) },
	} {
		var out, errb bytes.Buffer
		if code := run(&out, &errb); code != 1 {
			t.Errorf("%s: exit %d, want 1", name, code)
		}
		if out.Len() != 0 || errb.Len() == 0 {
			t.Errorf("%s: stdout %q, stderr %q; the load error belongs on stderr", name, out.String(), errb.String())
		}
	}
}

func TestScoreCountsWhitespaceNAReasonAsUnjustified(t *testing.T) {
	h, cells, _, err := Check(materialize(t, "na_whitespace_reason"))
	if err != nil {
		t.Fatal(err)
	}
	if m := computeMetrics(h, cells, 0); m.NAUnjustified != 1 {
		t.Fatalf("na_unjustified = %d, want 1", m.NAUnjustified)
	}
}

func renderJSON(t *testing.T, dir string) Roll {
	t.Helper()
	var out, errb bytes.Buffer
	Render(dir, true, false, &out, &errb)
	var r Roll
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("render --json: %v\n%s", err, out.String())
	}
	return r
}

func hasGap(gaps []string, id, reason string) bool {
	for _, g := range gaps {
		if strings.HasPrefix(g, id+":") && strings.Contains(g, reason) {
			return true
		}
	}
	return false
}

// A whitespace na_reason is no reason: the block must not read as a
// justified, fully covered N/A, and open gaps must name the cell.
func TestRenderWhitespaceNAReasonIsNotJustified(t *testing.T) {
	r := renderJSON(t, materialize(t, "na_whitespace_reason"))
	if got := r.GridV03["confidentiality"]["environment"]; got != "R 2/3" {
		t.Errorf("environment x confidentiality block = %q, want %q", got, "R 2/3")
	}
	if !hasGap(r.OpenGaps, "temporal.confidentiality", "na without reason") {
		t.Errorf("open gaps miss the unjustified na: %v", r.OpenGaps)
	}
}

func TestOpenGapsNameUnassignedOwnersAndOverdueRoadmaps(t *testing.T) {
	r := renderJSON(t, materialize(t, "open_gaps_unassigned_and_overdue"))
	if !hasGap(r.OpenGaps, "socio_legal.accountability", "owner unassigned") {
		t.Errorf("open gaps miss the unassigned owner: %v", r.OpenGaps)
	}
	if !hasGap(r.OpenGaps, "in_use.integrity", "roadmap overdue") {
		t.Errorf("open gaps miss the overdue roadmap: %v", r.OpenGaps)
	}
}

func TestOpenGapsAreNotCapped(t *testing.T) {
	dir := materialize(t, "cells_dir_missing") // 80 missing cells
	var md, html, errb bytes.Buffer
	Render(dir, false, false, &md, &errb)
	Render(dir, false, true, &html, &errb)
	if n := strings.Count(md.String(), "\n- "); n != 80 {
		t.Errorf("markdown lists %d gaps, want all 80", n)
	}
	if n := strings.Count(html.String(), "<li>"); n != 80 {
		t.Errorf("html lists %d gaps, want all 80", n)
	}
}

func TestInitWithoutTrailingSlashWritesIntoCellsDir(t *testing.T) {
	dir := materialize(t, "cells_dir_without_trailing_slash")
	var out, errb bytes.Buffer
	if code := Init(dir, &out, &errb); code != 0 {
		t.Fatalf("init exit %d: %s%s", code, out.String(), errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "cells", "operate.availability.yaml")); err != nil {
		t.Errorf("skeleton not in cells/: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "cellsoperate.availability.yaml")); err == nil {
		t.Errorf("skeleton written next to the cells dir, not inside it")
	}
}

func TestInitCreatesMissingCellsDir(t *testing.T) {
	dir := materialize(t, "cells_dir_missing")
	var out, errb bytes.Buffer
	if code := Init(dir, &out, &errb); code != 0 {
		t.Fatalf("init exit %d: %s%s", code, out.String(), errb.String())
	}
	ents, err := os.ReadDir(filepath.Join(dir, "matrix"))
	if err != nil || len(ents) != 80 {
		t.Fatalf("matrix/ has %d entries (err %v), want 80 skeletons", len(ents), err)
	}
}

// A cell that fails to parse is absent from the loaded grid, but its file
// is someone's work: init must never overwrite it with a skeleton.
func TestInitNeverOverwritesAnUnparseableCell(t *testing.T) {
	dir := materialize(t, "cell_unparseable")
	p := filepath.Join(dir, "cells", "at_rest.integrity.yaml")
	before, _ := os.ReadFile(p)
	var out, errb bytes.Buffer
	Init(dir, &out, &errb)
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatalf("init overwrote an existing cell file:\n%s", after)
	}
}
