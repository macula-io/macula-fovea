package core

import (
	"fmt"
	"html"
	"io"
	"time"
)

// The HTML scorecards target GitHub job summaries: GitHub sanitizes inline
// styles but renders tables and emoji, so the badges are emoji. Every piece
// of assessment text is escaped before it is written.

// renderV03HTML emits the v0.3 scorecard: emoji RAG badges with coverage
// counts, headline metrics, and the complete open-gaps list.
func renderV03HTML(w io.Writer, h *Header, cells map[string]*Cell, headerErrs []Finding, lintErrs int) {
	r := rollV03(h, cells, headerErrs, time.Now())
	m := computeMetrics(h, cells, lintErrs)

	fmt.Fprintln(w, "<h3>fovea scorecard: "+html.EscapeString(h.System)+" (spec v0.3)</h3>")
	gridHTML(w, h.AttributesVM(), r.GridV03)
	fmt.Fprintln(w, "<p>🟢 assessed · 🟡 assumed/roadmap · 🔴 unassessed/missing · ⚪ justified N/A · <code>n/total</code> = authored cells</p>")
	metricsHTML(w, m)
	if len(r.OpenGaps) == 0 {
		fmt.Fprintln(w, "<p>✅ <b>open gaps: none.</b> The grid is complete.</p>")
		return
	}
	fmt.Fprintf(w, "<details><summary><b>open gaps: %d</b></summary><ul>\n", len(r.OpenGaps))
	for _, g := range r.OpenGaps {
		fmt.Fprintln(w, "<li><code>"+html.EscapeString(g)+"</code></li>")
	}
	fmt.Fprintln(w, "</ul></details>")
}

// renderV02HTML emits the frozen v0.2 reading (a bare worst-RAG letter per
// block, plus the per-family roll-up) as HTML, so the job summary gets the
// same kind of document whichever spec version the header declares.
func renderV02HTML(w io.Writer, h *Header, cells map[string]*Cell, lintErrs int) {
	r := rollV02(h, cells)
	m := computeMetrics(h, cells, lintErrs)

	fmt.Fprintln(w, "<h3>fovea scorecard: "+html.EscapeString(h.System)+" (spec v0.2)</h3>")
	gridHTML(w, h.AttributesVM(), r.Grid)
	fmt.Fprintf(w, "<p><b>rollup</b> actors %s · lifecycle %s · data %s · environment %s</p>\n",
		badge(r.ByFamily["actors"]), badge(r.ByFamily["lifecycle"]), badge(r.ByFamily["data"]), badge(r.ByFamily["environment"]))
	fmt.Fprintln(w, "<p>🟢 assessed · 🟡 assumed/roadmap · 🔴 unassessed/missing · ⚪ N/A; each block shows its worst cell</p>")
	metricsHTML(w, m)
}

func gridHTML(w io.Writer, attrs []string, grid map[string]map[string]string) {
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(w, "<tr><th>attribute</th><th>actors</th><th>lifecycle</th><th>data</th><th>environment</th></tr>")
	for _, a := range attrs {
		g := grid[a]
		fmt.Fprintf(w, "<tr><td><code>%s</code></td>%s%s%s%s</tr>\n",
			html.EscapeString(a), cellHTML(g["actors"]), cellHTML(g["lifecycle"]), cellHTML(g["data"]), cellHTML(g["environment"]))
	}
	fmt.Fprintln(w, "</table>")
}

func metricsHTML(w io.Writer, m Metrics) {
	fmt.Fprintf(w, "<p><b>lint errors</b> %d · <b>unassessed</b> %.1f%% · <b>na unjustified</b> %d · <b>by-design measures</b> %.1f%% · <b>overdue roadmaps</b> %d</p>\n",
		m.LintErrors, 100*m.PctUnassessed, m.NAUnjustified, 100*m.PctByDesign, m.OverdueRoadmap)
}

// badge maps a RAG letter to its emoji.
func badge(rag string) string {
	switch rag {
	case ragGreen:
		return "🟢"
	case ragAmber:
		return "🟡"
	case ragRed:
		return "🔴"
	case ragGrey:
		return "⚪"
	default:
		return "·"
	}
}

// cellHTML renders one block: a v0.2 block is a bare RAG letter ("R"), a
// v0.3 block a letter plus coverage ("R 2/6", or "- 3/3" for all N/A).
func cellHTML(s string) string {
	if s == "·" || s == "" {
		return "<td align=\"center\">·</td>"
	}
	if len(s) == 1 {
		return "<td align=\"center\">" + badge(s) + "</td>"
	}
	return "<td align=\"center\">" + badge(s[:1]) + " <code>" + html.EscapeString(s[2:]) + "</code></td>"
}
