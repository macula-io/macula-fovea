package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
)

type Roll struct {
	ByAttr      map[string]string            `json:"by_attribute"`
	ByFamily    map[string]string            `json:"by_family"`
	Grid        map[string]map[string]string `json:"grid"`      // attribute -> family -> rag (0.2)
	GridV03     map[string]map[string]string `json:"grid_v03"`  // attribute -> family -> "RAG n/total" (0.3)
	Coverage    map[string]map[string]string `json:"coverage"`  // attribute -> family -> "authored/total"
	OpenGaps    []string                     `json:"open_gaps"` // v0.3: named gaps
	MetricsNote string                       `json:"note"`
	Version     string                       `json:"version"`
}

// Render prints the scorecard. 0.2 headers get the frozen grid; 0.3 headers
// get the coverage-aware grid plus the open-gaps section (spec v0.3, 13).
// --html emits GitHub-job-summary-native HTML (emoji RAG badges; inline
// styles are sanitized by GitHub, emoji is not).
//
// The scorecard is written even when lint fails, because a red run should
// leave its report behind; but the findings go to stderr and the exit code
// is 1, so a lint-failing assessment never renders as a clean result.
func Render(dir string, jsonOut, htmlOut bool, stdout, stderr io.Writer) int {
	h, cells, fs, err := Check(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var headerErrs, errs []Finding
	for _, f := range fs {
		if !f.Err {
			continue
		}
		errs = append(errs, f)
		if f.Where == "fovea.yaml" {
			headerErrs = append(headerErrs, f)
		}
	}

	switch {
	case h.Fovea == "0.3" && htmlOut && !jsonOut:
		renderV03HTML(stdout, h, cells, headerErrs, len(errs))
	case h.Fovea == "0.3":
		renderV03(stdout, h, cells, headerErrs, jsonOut)
	case htmlOut && !jsonOut:
		renderV02HTML(stdout, h, cells, len(errs))
	default:
		renderV02(stdout, h, cells, jsonOut)
	}

	if len(errs) > 0 {
		printFindings(stderr, errs)
		fmt.Fprintf(stderr, "note: %d lint error(s); the scorecard above is not a final reading. Run: fovea lint %s\n", len(errs), dir)
		return 1
	}
	return 0
}

// ---- 0.2: the frozen reading (unchanged output, worst-RAG per block) ----

func rollV02(h *Header, cells map[string]*Cell) Roll {
	r := Roll{Version: "0.2", ByAttr: map[string]string{}, ByFamily: map[string]string{}, Grid: map[string]map[string]string{}}
	attrs := h.AttributesVM()
	cols := h.DeclaredColumns()

	for _, a := range attrs {
		row := map[string]string{}
		for _, fam := range families {
			worst := -1
			anyCol := false
			for _, c := range cols {
				if h.FamilyOf(c) != fam {
					continue
				}
				anyCol = true
				if r := ragRank(cellRAG(cells[c+"."+a])); r > worst {
					worst = r
				}
			}
			if !anyCol {
				row[fam] = "·"
			} else {
				row[fam] = ragLetter(worst)
			}
		}
		r.Grid[a] = row
	}
	fillRollups(&r, attrs, r.Grid)
	return r
}

func renderV02(w io.Writer, h *Header, cells map[string]*Cell, jsonOut bool) {
	r := rollV02(h, cells)
	attrs := h.AttributesVM()

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}

	fmt.Fprintf(w, "# Scorecard: %s\n\n", h.System)
	fmt.Fprintln(w, "| attribute     | actors | lifecycle | data | environment |")
	fmt.Fprintln(w, "|---|---|---|---|---|")
	for _, a := range attrs {
		g := r.Grid[a]
		fmt.Fprintf(w, "| %-13s | %s | %s | %s | %s |\n", a, g["actors"], g["lifecycle"], g["data"], g["environment"])
	}
	fmt.Fprintln(w, "| **rollup**    | "+r.ByFamily["actors"]+" | "+r.ByFamily["lifecycle"]+" | "+r.ByFamily["data"]+" | "+r.ByFamily["environment"]+" |")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "R=Red(unassessed/missing), A=Amber(assumed/roadmap), -=Grey(na), G=Green(assessed); block = cell with worst RAG in the block")
}

// ---- 0.3: coverage-aware grid + open gaps; worst-RAG is the invariant ----

// cellRAGv03 is cellRAG with spec v0.3's N/A rule: only a justified na is
// grey. An na without a written reason is unfinished work and stays red.
func cellRAGv03(c *Cell) string {
	if c != nil && c.Status == "na" && !naJustified(c) {
		return ragRed
	}
	return cellRAG(c)
}

// authored reports whether a cell counts toward a block's coverage: it
// exists and is answered. unassessed and unjustified na are not answers.
func authored(c *Cell) bool {
	return c != nil && c.Status != "unassessed" && (c.Status != "na" || naJustified(c))
}

// rollV03 computes the v0.3 grid, coverage and open gaps. Markdown, JSON
// and HTML all render this one value, so they cannot disagree.
func rollV03(h *Header, cells map[string]*Cell, headerErrs []Finding, today time.Time) Roll {
	r := Roll{
		Version:  "0.3",
		ByAttr:   map[string]string{},
		ByFamily: map[string]string{},
		GridV03:  map[string]map[string]string{},
		Coverage: map[string]map[string]string{},
	}
	attrs := h.AttributesVM()
	cols := h.DeclaredColumns()

	for _, a := range attrs {
		row := map[string]string{}
		cov := map[string]string{}
		for _, fam := range families {
			worst := -1
			done, total := 0, 0
			for _, c := range cols {
				if h.FamilyOf(c) != fam {
					continue
				}
				total++
				cell := cells[c+"."+a]
				if authored(cell) {
					done++
				}
				rg := cellRAGv03(cell)
				if rg == ragGrey {
					continue // justified N/A: no threat surface, must not drag the block down
				}
				if rk := ragRank(rg); rk > worst {
					worst = rk
				}
			}
			switch {
			case total == 0:
				row[fam] = "·"
				cov[fam] = "·"
			case worst == -1:
				// every cell in the block is a justified N/A
				row[fam] = fmt.Sprintf("- %d/%d", done, total)
				cov[fam] = fmt.Sprintf("%d/%d", done, total)
			default:
				// Worst-RAG stays: one unfinished cell keeps the block red.
				row[fam] = fmt.Sprintf("%s %d/%d", ragLetter(worst), done, total)
				cov[fam] = fmt.Sprintf("%d/%d", done, total)
			}
		}
		r.GridV03[a] = row
		r.Coverage[a] = cov
	}
	fillRollups(&r, attrs, r.GridV03)
	r.OpenGaps = openGaps(h, cells, headerErrs, today)
	r.MetricsNote = "RAG = worst cell in the block (invariant); n/total = authored cells; - = all cells justified N/A. Open gaps listed below carry the unfinished work."
	return r
}

func renderV03(w io.Writer, h *Header, cells map[string]*Cell, headerErrs []Finding, jsonOut bool) {
	r := rollV03(h, cells, headerErrs, time.Now())
	attrs := h.AttributesVM()

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}

	fmt.Fprintf(w, "# Scorecard: %s (spec v0.3)\n\n", h.System)
	fmt.Fprintln(w, "| attribute     | actors | lifecycle | data | environment |")
	fmt.Fprintln(w, "|---|---|---|---|---|")
	for _, a := range attrs {
		g := r.GridV03[a]
		fmt.Fprintf(w, "| %-13s | %-7s | %-10s | %-5s | %-12s |\n", a, g["actors"], g["lifecycle"], g["data"], g["environment"])
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, r.MetricsNote)
	fmt.Fprintln(w)
	printOpenGaps(w, r.OpenGaps)
}

func fillRollups(r *Roll, attrs []string, grid map[string]map[string]string) {
	for _, a := range attrs {
		w := -1
		for _, fam := range families {
			x := grid[a][fam]
			if x == "·" {
				continue
			}
			fr := firstRag(x)
			if fr == "-" {
				continue // all-N/A block: nothing assessable, roll up from what exists
			}
			if t := ragRank(fr); t > w {
				w = t
			}
		}
		r.ByAttr[a] = ragLetter(w)
	}
	for _, fam := range families {
		w := -1
		for _, a := range attrs {
			x := grid[a][fam]
			if x == "·" {
				continue
			}
			fr := firstRag(x)
			if fr == "-" {
				continue
			}
			if t := ragRank(fr); t > w {
				w = t
			}
		}
		r.ByFamily[fam] = ragLetter(w)
	}
}

// firstRag takes the RAG letter from a v0.3 cell string like "R 1/6".
func firstRag(s string) string {
	for _, r := range s {
		if r == ' ' {
			break
		}
		return string(r)
	}
	return "·"
}

// openGaps names the unfinished work (spec v0.3, 13): header lint errors
// (an unassigned header owner among them), missing cells, unassessed
// cells, unassigned cell owners, unjustified NAs and overdue roadmaps. Each
// entry reads "<where>: <reason>". The list is complete; it is what CI
// archives, so nothing is cut.
func openGaps(h *Header, cells map[string]*Cell, headerErrs []Finding, today time.Time) []string {
	var gaps []string
	for _, f := range headerErrs {
		gaps = append(gaps, "fovea.yaml: "+f.Message)
	}
	for _, id := range h.ExpectedCells() {
		c, ok := cells[id]
		if !ok {
			gaps = append(gaps, id+": missing cell")
			continue
		}
		if c.Status == "unassessed" {
			gaps = append(gaps, id+": unassessed")
		}
		if unassigned(c.Owner) {
			gaps = append(gaps, id+": owner unassigned")
		}
		if c.Status == "na" && !naJustified(c) {
			gaps = append(gaps, id+": na without reason")
		}
		if c.Status == "roadmap" && isOverdue(c, today) {
			gaps = append(gaps, fmt.Sprintf("%s: roadmap overdue (review_by %s)", id, c.ReviewBy))
		}
	}
	sort.Strings(gaps)
	return gaps
}

func printOpenGaps(w io.Writer, gaps []string) {
	if len(gaps) == 0 {
		fmt.Fprintln(w, "## Open gaps: none. The grid is complete.")
		return
	}
	fmt.Fprintf(w, "## Open gaps: %d\n", len(gaps))
	for _, g := range gaps {
		fmt.Fprintf(w, "- %s\n", g)
	}
}

func ragLetter(rank int) string {
	switch rank {
	case -1:
		return "·"
	case 0:
		return ragGreen
	case 1:
		return ragGrey
	case 2:
		return ragAmber
	default:
		return ragRed
	}
}
