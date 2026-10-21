package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	ragGreen = "G"
	ragAmber = "A"
	ragRed   = "R"
	ragGrey  = "-"
)

// ragRank orders severity for roll-ups: Red > Amber > Grey > Green.
func ragRank(r string) int {
	switch r {
	case ragRed:
		return 3
	case ragAmber:
		return 2
	case ragGrey:
		return 1
	default:
		return 0
	}
}

// cellRAG implements the RAG mapping in spec 13-scorecard. Missing cells
// count as unassessed (Red): the grid closure rule, scored.
func cellRAG(c *Cell) string {
	if c == nil {
		return ragRed
	}
	switch c.Status {
	case "assessed":
		return ragGreen
	case "assumed":
		return ragAmber
	case "roadmap":
		return ragAmber
	case "na":
		return ragGrey
	default:
		return ragRed
	}
}

type Metrics struct {
	System         string         `json:"system"`
	ExpectedCells  int            `json:"expected_cells"`
	PresentCells   int            `json:"present_cells"`
	MissingCells   int            `json:"missing_cells"`
	PctUnassessed  float64        `json:"pct_unassessed"`
	NAUnjustified  int            `json:"na_unjustified"`
	PctByDesign    float64        `json:"pct_by_design"`
	OldestReviewBy string         `json:"oldest_review_by,omitempty"`
	OverdueRoadmap int            `json:"overdue_roadmap"`
	ByAttribute    map[string]int `json:"cells_by_attribute"`
	ByFamily       map[string]int `json:"cells_by_family"`
	LintErrors     int            `json:"lint_errors"`
}

// Score runs coverage-aware scoring (13-scorecard). Exit 1 on lint errors.
func Score(dir string, jsonOut bool, stdout, stderr io.Writer) int {
	h, cells, lf, err := Check(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	lintErrs := 0
	for _, f := range lf {
		if f.Err {
			lintErrs++
		}
	}

	m := computeMetrics(h, cells, lintErrs)

	if jsonOut {
		b, _ := json.MarshalIndent(m, "", "  ")
		fmt.Fprintln(stdout, string(b))
	} else {
		printScore(stdout, m)
	}
	if lintErrs > 0 {
		// stderr, never stdout: `score --json > score.json` must stay JSON
		// on the runs where it matters most, the failing ones.
		fmt.Fprintf(stderr, "note: %d lint error(s); run: fovea lint %s\n", lintErrs, dir)
		return 1
	}
	return 0
}

func computeMetrics(h *Header, cells map[string]*Cell, lintErrs int) Metrics {
	m := Metrics{
		System:        h.System,
		ByAttribute:   map[string]int{},
		ByFamily:      map[string]int{},
		LintErrors:    lintErrs,
		ExpectedCells: len(h.ExpectedCells()),
	}
	t := cellTally{today: time.Now()}
	for _, id := range h.ExpectedCells() {
		t.add(h, id, cells[id], m.ByAttribute, m.ByFamily)
	}
	m.PresentCells = len(cells)
	m.MissingCells = t.total - presentExpected(h, cells)
	m.PctUnassessed = pct(t.unassessed, t.total)
	m.NAUnjustified = t.naUnjustified
	m.PctByDesign = pct(t.byDesign, t.totalMeasures)
	m.OldestReviewBy = t.oldest
	m.OverdueRoadmap = t.overdue
	return m
}

// cellTally accumulates the score metrics across the expected grid.
type cellTally struct {
	today         time.Time
	total         int
	unassessed    int
	naUnjustified int
	byDesign      int
	totalMeasures int
	overdue       int
	oldest        string
}

func (t *cellTally) add(h *Header, id string, c *Cell, byAttr, byFam map[string]int) {
	t.total++
	if c == nil || c.Status == "unassessed" {
		t.unassessed++
	}
	if c == nil {
		return
	}
	t.tallyAnswer(c)
	t.tallyPosition(h, id, byAttr, byFam)
}

// tallyAnswer counts what the cell says: unjustified N/A, measures, and a
// roadmap review date that is due.
func (t *cellTally) tallyAnswer(c *Cell) {
	if c.Status == "na" && !naJustified(c) {
		t.naUnjustified++
	}
	t.tallyReviewBy(c)
	for _, ms := range allMeasures(c) {
		t.totalMeasures++
		if ms.Status == "by_design" {
			t.byDesign++
		}
	}
}

func (t *cellTally) tallyReviewBy(c *Cell) {
	if c.Status != "roadmap" || c.ReviewBy == "" {
		return
	}
	due, err := time.Parse("2006-01-02", c.ReviewBy)
	if err != nil {
		return
	}
	if t.oldest == "" || due.Before(mustParse(t.oldest)) {
		t.oldest = c.ReviewBy
	}
	if due.Before(t.today) {
		t.overdue++
	}
}

// tallyPosition counts the cell under its column and its attribute family.
func (t *cellTally) tallyPosition(h *Header, id string, byAttr, byFam map[string]int) {
	attr := ""
	if i := strings.LastIndex(id, "."); i >= 0 {
		attr = id[i+1:]
	}
	byAttr[attr]++
	byFam[h.FamilyOf(id[:len(id)-len(attr)-1])]++
}

// presentExpected counts how many declared-grid ids were actually provided.
func presentExpected(h *Header, cells map[string]*Cell) int {
	n := 0
	for _, id := range h.ExpectedCells() {
		if _, ok := cells[id]; ok {
			n++
		}
	}
	return n
}

func mustParse(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func printScore(w io.Writer, m Metrics) {
	fmt.Fprintf(w, "fovea score: %s\n", m.System)
	fmt.Fprintf(w, "  expected cells      %d\n", m.ExpectedCells)
	fmt.Fprintf(w, "  present cells       %d (missing %d)\n", m.PresentCells, m.MissingCells)
	fmt.Fprintf(w, "  pct_unassessed      %.1f%%\n", 100*m.PctUnassessed)
	fmt.Fprintf(w, "  na_unjustified      %d\n", m.NAUnjustified)
	fmt.Fprintf(w, "  pct_by_design       %.1f%%\n", 100*m.PctByDesign)
	if m.OldestReviewBy != "" {
		fmt.Fprintf(w, "  oldest review_by    %s (%d overdue)\n", m.OldestReviewBy, m.OverdueRoadmap)
	}
	fmt.Fprintln(w, "  cells per attribute")
	attrs := make([]string, 0, len(m.ByAttribute))
	for a := range m.ByAttribute {
		attrs = append(attrs, a)
	}
	sort.Strings(attrs)
	for _, a := range attrs {
		fmt.Fprintf(w, "    %-16s %d\n", a, m.ByAttribute[a])
	}
	fmt.Fprintln(w, "  cells per family")
	for _, f := range families {
		fmt.Fprintf(w, "    %-16s %d\n", f, m.ByFamily[f])
	}
}
