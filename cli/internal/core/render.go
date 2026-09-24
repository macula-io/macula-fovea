package core

import (
	"encoding/json"
	"fmt"
	"sort"
)

type Roll struct {
	ByAttr      map[string]string            `json:"by_attribute"`
	ByFamily    map[string]string            `json:"by_family"`
	Grid        map[string]map[string]string `json:"grid"`        // attribute -> family -> rag (0.2)
	GridV03     map[string]map[string]string `json:"grid_v03"`    // attribute -> family -> "RAG n/total" (0.3)
	Coverage    map[string]map[string]string `json:"coverage"`    // attribute -> family -> "authored/total"
	OpenGaps    []string                     `json:"open_gaps"`   // v0.3: named gaps
	MetricsNote string                       `json:"note"`
	Version     string                       `json:"version"`
}

// Render prints the scorecard. 0.2 headers get the frozen grid; 0.3 headers
// get the coverage-aware grid plus the open-gaps section (spec v0.3, 13).
func Render(dir string, jsonOut bool) int {
	h, _, err := LoadHeader(dir)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	cells, _, _ := LoadCells(dir, h)

	if h.Fovea == "0.3" {
		return renderV03(h, cells, jsonOut)
	}
	return renderV02(h, cells, jsonOut)
}

// ---- 0.2: the frozen reading (unchanged output, worst-RAG per block) ----

func renderV02(h *Header, cells map[string]*Cell, jsonOut bool) int {
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

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	fmt.Printf("# Scorecard — %s\n\n", h.System)
	fmt.Println("| attribute     | actors | lifecycle | data | environment |")
	fmt.Println("|---|---|---|---|---|")
	for _, a := range attrs {
		g := r.Grid[a]
		fmt.Printf("| %-13s | %s | %s | %s | %s |\n", a, g["actors"], g["lifecycle"], g["data"], g["environment"])
	}
	fmt.Println("| **rollup**    | " + r.ByFamily["actors"] + " | " + r.ByFamily["lifecycle"] + " | " + r.ByFamily["data"] + " | " + r.ByFamily["environment"] + " |")
	fmt.Println()
	fmt.Println("R=Red(unassessed/missing), A=Amber(assumed/roadmap), -=Grey(na), G=Green(assessed); block = cell with worst RAG in the block")
	return 0
}

// ---- 0.3: coverage-aware grid + open gaps; worst-RAG is the invariant ----

func renderV03(h *Header, cells map[string]*Cell, jsonOut bool) int {
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
			authored, total := 0, 0
			anyCol := false
			for _, c := range cols {
				if h.FamilyOf(c) != fam {
					continue
				}
				anyCol = true
				total++
				id := c + "." + a
				cell := cells[id]
				if cell != nil && cell.Status != "unassessed" {
					authored++
				}
				if r := ragRank(cellRAG(cell)); r > worst {
					worst = r
				}
			}
			if !anyCol {
				row[fam] = "·"
				cov[fam] = "·"
			} else {
				// Worst-RAG stays: one unassessed/missing cell keeps the block red.
				row[fam] = fmt.Sprintf("%s %d/%d", ragLetter(worst), authored, total)
				cov[fam] = fmt.Sprintf("%d/%d", authored, total)
			}
		}
		r.GridV03[a] = row
		r.Coverage[a] = cov
	}
	fillRollups(&r, attrs, r.GridV03)
	r.OpenGaps = openGaps(h, cells)
	r.MetricsNote = "RAG = worst cell in the block (invariant); n/total = authored cells. Open gaps listed below carry the unfinished work."

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	fmt.Printf("# Scorecard — %s (spec v0.3)\n\n", h.System)
	fmt.Println("| attribute     | actors | lifecycle | data | environment |")
	fmt.Println("|---|---|---|---|---|")
	for _, a := range attrs {
		g := r.GridV03[a]
		fmt.Printf("| %-13s | %-7s | %-10s | %-5s | %-12s |\n", a, g["actors"], g["lifecycle"], g["data"], g["environment"])
	}
	fmt.Println()
	fmt.Println(r.MetricsNote)
	fmt.Println()
	printOpenGaps(r.OpenGaps)
	return 0
}

func fillRollups(r *Roll, attrs []string, grid map[string]map[string]string) {
	for _, a := range attrs {
		w := -1
		for _, fam := range families {
			x := grid[a][fam]
			if x == "·" {
				continue
			}
			if t := ragRank(firstRag(x)); t > w {
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
			if t := ragRank(firstRag(x)); t > w {
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

// openGaps names the unfinished work: missing cells, unassigned cells,
// unjustified NAs, overdue roadmaps. The gap-info formerly implied by a red
// wall, now stated as a list (spec v0.3).
func openGaps(h *Header, cells map[string]*Cell) []string {
	var gaps []string
	for _, id := range h.ExpectedCells() {
		c, ok := cells[id]
		if !ok {
			gaps = append(gaps, fmt.Sprintf("%s — missing cell", id))
			continue
		}
		switch c.Status {
		case "unassessed":
			gaps = append(gaps, fmt.Sprintf("%s — unassessed", id))
		case "na":
			if c.NAReason == "" {
				gaps = append(gaps, fmt.Sprintf("%s — na without reason", id))
			}
		}
	}
	sort.Strings(gaps)
	return gaps
}

func printOpenGaps(gaps []string) {
	if len(gaps) == 0 {
		fmt.Println("## Open gaps — none. The grid is complete.")
		return
	}
	fmt.Printf("## Open gaps — %d\n", len(gaps))
	const capN = 40
	for i, g := range gaps {
		if i == capN {
			fmt.Printf("… and %d more\n", len(gaps)-capN)
			break
		}
		fmt.Printf("- %s\n", g)
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
