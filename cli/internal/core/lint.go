package core

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var statuses = map[string]bool{
	"unassessed": true, "assumed": true, "assessed": true, "roadmap": true, "na": true,
}

var measureStatuses = map[string]bool{"by_design": true, "roadmap": true, "org": true}

func allMeasures(c *Cell) (out []Measure) {
	out = append(out, c.Defense.Detection...)
	out = append(out, c.Defense.Countermeasures...)
	out = append(out, c.Defense.Recovery...)
	return
}

// Lint enforces the anti-theater and cell rules of spec v0.2 and v0.3 (00-overview,
// 12-cell-schema, 13-scorecard) against an assessment directory.
func Lint(dir string, github bool, stdout, stderr io.Writer) int {
	h, _, f, err := Check(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if github {
		return lintGithub(stdout, dir, h, f)
	}

	fmt.Fprintf(stdout, "fovea lint: %s (%d cells expected)\n", h.System, len(h.ExpectedCells()))
	if len(f) == 0 {
		fmt.Fprintln(stdout, "  clean: no findings")
		return 0
	}
	errs, warns := printFindings(stdout, f)
	fmt.Fprintf(stdout, "  %d error(s), %d warning(s)\n", errs, warns)
	if errs > 0 {
		return 1
	}
	return 0
}

// Check loads an assessment directory and runs every lint rule against it.
// It is the single entry point lint, score and render share, so the three
// commands can never disagree about what an assessment is.
func Check(dir string) (*Header, map[string]*Cell, []Finding, error) {
	h, f, err := LoadHeader(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	cells, _, cf := LoadCells(dir, h)
	f = append(f, cf...)
	f = append(f, lintCoverage(h, cells)...)
	f = append(f, lintEachCell(h, cells)...)
	f = append(f, lintDuplicates(cells)...)
	return h, cells, f, nil
}

// lintGithub emits GitHub Actions workflow commands so failures annotate the
// offending cell file in the PR diff. `dir` is the assessment directory as
// passed by the caller (e.g. "security/fovea"); annotation paths are built
// relative to the repository root.
func lintGithub(w io.Writer, dir string, h *Header, fs []Finding) int {
	cellsDir := strings.TrimSuffix(h.CellsDir, "/")
	errs := 0
	for _, f := range fs {
		where := f.Where
		var path string
		switch {
		case where == "fovea.yaml":
			path = filepath.Join(dir, "fovea.yaml")
		case strings.HasSuffix(where, ".yaml"):
			path = filepath.Join(dir, cellsDir, where)
		default:
			path = filepath.Join(dir, cellsDir, where+".yaml")
		}
		cmd := "warning"
		if f.Err {
			cmd = "error"
			errs++
		}
		fmt.Fprintf(w, "::%s file=%s,line=1,title=fovea::%s\n", cmd, path, f.Message)
	}
	if errs > 0 {
		return 1
	}
	return 0
}

// lintCoverage implements "the grid is closing" (00-overview rule 1).
func lintCoverage(h *Header, cells map[string]*Cell) []Finding {
	var f []Finding
	for _, id := range h.ExpectedCells() {
		if _, ok := cells[id]; !ok {
			f = append(f, errf("grid_missing_cell", id, "missing cell (run: fovea init)"))
		}
	}
	return f
}

func lintEachCell(h *Header, cells map[string]*Cell) []Finding {
	var f []Finding
	declaredCol := map[string]bool{}
	for _, c := range h.DeclaredColumns() {
		declaredCol[c] = true
	}
	declaredAttr := map[string]bool{}
	for _, a := range h.AttributesVM() {
		declaredAttr[a] = true
	}

	for id, c := range cells {
		// Rule: no invisible columns/attributes (12-cell-schema hard rule 2).
		parts := strings.SplitN(id, ".", 2)
		if len(parts) != 2 || !declaredCol[parts[0]] || !declaredAttr[parts[1]] {
			f = append(f, errf("cell_id_undeclared", id, "id parts not declared in the header"))
		}
		if !statuses[c.Status] {
			f = append(f, errf("cell_status_unknown", id, "unknown status %q", c.Status))
		}
		if unassigned(c.Owner) {
			f = append(f, errf("cell_owner_unassigned", id, "owner is unassigned"))
		}
		switch c.Status {
		case "unassessed":
			// Spec 00 status table: unassessed is not allowed at final, and
			// lint is what "final" means. A present cell must be answered.
			f = append(f, errf("cell_unassessed", id, "status unassessed: the cell exists but nobody has answered it"))
		case "na":
			if !naJustified(c) {
				f = append(f, errf("na_without_reason", id, "status na without na_reason"))
			}
		case "roadmap", "assumed", "assessed":
			// Content rules hold for every answered cell, roadmap included:
			// a roadmap cell is a work package and must say what the work is.
			if strings.TrimSpace(c.Threat.Definition) == "" {
				f = append(f, errf("definition_empty", id, "empty threat definition"))
			}
			if len(c.Threat.Manifestations) == 0 {
				f = append(f, errf("manifestations_empty", id, "zero manifestations"))
			}
		}
		hasRoadmapMeasure := false
		for _, m := range allMeasures(c) {
			hasRoadmapMeasure = hasRoadmapMeasure || m.Status == "roadmap"
		}
		switch {
		case strings.TrimSpace(c.ReviewBy) != "":
			if _, err := time.Parse("2006-01-02", c.ReviewBy); err != nil {
				f = append(f, errf("review_by_not_iso_date", id, "review_by %q is not an ISO date", c.ReviewBy))
			}
		case c.Status == "roadmap":
			f = append(f, errf("roadmap_without_review_by", id, "status roadmap without review_by"))
		case hasRoadmapMeasure:
			// Spec 12 measure statuses: a roadmap measure requires review_by
			// on the cell, whatever the cell's own status.
			f = append(f, errf("roadmap_measure_without_review_by", id, "roadmap measure without review_by on the cell"))
		}
		for _, m := range allMeasures(c) {
			if !measureStatuses[m.Status] {
				f = append(f, errf("measure_status_unknown", id, "measure with unknown status %q", m.Status))
			}
			if m.Status == "by_design" && strings.TrimSpace(m.Source) == "" {
				f = append(f, errf("by_design_without_source", id, "by_design measure without source"))
			}
		}
		// Anti-theater: assessed must mean the product answers something.
		if c.Status == "assessed" {
			ms := allMeasures(c)
			if len(ms) > 0 {
				allOrg := true
				for _, m := range ms {
					if m.Status != "org" {
						allOrg = false
					}
				}
				if allOrg {
					f = append(f, errf("assessed_all_org", id, "status assessed but every measure is org; the cell is org-managed, not assessed"))
				}
			}
		}
	}
	return f
}

// naJustified is the one test of "na with a written reason" that lint,
// score and render all use: whitespace is not a reason.
func naJustified(c *Cell) bool {
	return strings.TrimSpace(c.NAReason) != ""
}

// lintDuplicates flags cells whose threat definitions look copy-pasted
// (00-overview anti-theater rule: copy-paste is the primary symptom).
func lintDuplicates(cells map[string]*Cell) []Finding {
	var f []Finding
	toks := map[string]map[string]bool{}
	for id, c := range cells {
		toks[id] = tokenSet(c.Threat.Definition)
	}
	ids := make([]string, 0, len(cells))
	for id := range cells {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			sim := jaccard(toks[ids[i]], toks[ids[j]])
			if sim >= 0.85 {
				f = append(f, errf("definition_copy_paste", ids[i], "threat definition ≥85%% similar to %s (%.2f), copy-paste", ids[j], sim))
			}
		}
	}
	return f
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	f := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return ' '
	}
	for _, w := range strings.Fields(strings.Map(f, s)) {
		if len(w) >= 3 {
			out[w] = true
		}
	}
	return out
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}
