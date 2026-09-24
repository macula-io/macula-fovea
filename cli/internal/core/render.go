package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Roll struct {
	ByAttr      map[string]string            `json:"by_attribute"`
	ByFamily    map[string]string            `json:"by_family"`
	Grid        map[string]map[string]string `json:"grid"` // attribute -> family -> rag
	MetricsNote string                       `json:"note"`
}

// Render prints the scorecard as a markdown grid: attributes as rows, the
// four column families as columns; cell content is the *worst* RAG among
// that block's cells (Red > Amber > Grey > Green), with "·" where there is
// no declared column in that family.
func Render(dir string, jsonOut bool) int {
	h, _, err := LoadHeader(dir)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	cells, _, _ := LoadCells(dir, h)

	r := Roll{
		ByAttr:   map[string]string{},
		ByFamily: map[string]string{},
		Grid:     map[string]map[string]string{},
	}

	// Grid: attribute × family -> worst RAG.
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
				rg := cellRAG(cells[c+"."+a])
				if r := ragRank(rg); r > worst {
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

	// Roll-ups.
	for _, a := range attrs {
		w := -1
		for _, fam := range families {
			x := r.Grid[a][fam]
			if x == "·" {
				continue
			}
			if t := ragRank(x); t > w {
				w = t
			}
		}
		r.ByAttr[a] = ragLetter(w)
	}
	for _, fam := range families {
		w := -1
		for _, a := range attrs {
			x := r.Grid[a][fam]
			if x == "·" {
				continue
			}
			if t := ragRank(x); t > w {
				w = t
			}
		}
		r.ByFamily[fam] = ragLetter(w)
	}
	r.MetricsNote = "R=Red(unassessed/missing), A=Amber(assumed/roadmap), -=Grey(na), G=Green(assessed); block = cell with worst RAG in the block"

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
		fmt.Printf("| %-13s | %s | %s | %s | %s |\n",
			a, pad(g["actors"], 6), pad(g["lifecycle"], 9), pad(g["data"], 4), pad(g["environment"], 11))
	}
	fmt.Println("| **rollup**    | " + pad(r.ByFamily["actors"], 6) + " | " + pad(r.ByFamily["lifecycle"], 9) + " | " + pad(r.ByFamily["data"], 4) + " | " + pad(r.ByFamily["environment"], 11) + " |")
	fmt.Println()
	fmt.Println(r.MetricsNote)
	return 0
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

func pad(s string, _ int) string {
	if s == "" {
		return " "
	}
	if strings.HasPrefix(s, "%") {
		return " "
	}
	return s
}
