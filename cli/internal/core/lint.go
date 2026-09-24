package core

import (
	"fmt"
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

// Lint enforces spec v0.2's anti-theater and schema rules (00-overview,
// 12-cell-schema, 13-scorecard) against an assessment directory.
func Lint(dir string) int {
	h, f, err := LoadHeader(dir)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	cells, _, cf := LoadCells(dir, h)
	f = append(f, cf...)

	f = append(f, lintCoverage(h, cells)...)
	f = append(f, lintEachCell(h, cells)...)
	f = append(f, lintDuplicates(cells)...)

	fmt.Printf("fovea lint — %s (%d cells expected)\n", h.System, len(h.ExpectedCells()))
	if len(f) == 0 {
		fmt.Println("  clean — no findings")
		return 0
	}
	errs, warns := printFindings(f)
	fmt.Printf("  %d error(s), %d warning(s)\n", errs, warns)
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
			f = append(f, errf(id, "missing cell (run: fovea init)"))
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
			f = append(f, errf(id, "id parts not declared in the header"))
		}
		if !statuses[c.Status] {
			f = append(f, errf(id, "unknown status %q", c.Status))
		}
		if c.Owner == "" || c.Owner == "unassigned" {
			f = append(f, errf(id, "owner is unassigned"))
		}
		switch c.Status {
		case "na":
			if strings.TrimSpace(c.NAReason) == "" {
				f = append(f, errf(id, "status na without na_reason"))
			}
		case "roadmap":
			if c.ReviewBy == "" {
				f = append(f, errf(id, "status roadmap without review_by"))
			} else if _, err := time.Parse("2006-01-02", c.ReviewBy); err != nil {
				f = append(f, errf(id, "review_by %q is not an ISO date", c.ReviewBy))
			}
		default:
			if strings.TrimSpace(c.Threat.Definition) == "" {
				f = append(f, errf(id, "empty threat definition"))
			}
			if len(c.Threat.Manifestations) == 0 {
				f = append(f, errf(id, "zero manifestations"))
			}
		}
		for _, m := range allMeasures(c) {
			if !measureStatuses[m.Status] {
				f = append(f, errf(id, "measure with unknown status %q", m.Status))
			}
			if m.Status == "by_design" && strings.TrimSpace(m.Source) == "" {
				f = append(f, errf(id, "by_design measure without source"))
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
					f = append(f, errf(id, "status assessed but every measure is org — cell is org-managed, not assessed"))
				}
			}
		}
	}
	return f
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
				f = append(f, errf(ids[i], "threat definition ≥85%% similar to %s (%.2f) — copy-paste", ids[j], sim))
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
