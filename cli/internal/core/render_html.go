package core

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// renderV03HTML emits a GitHub-job-summary-native HTML scorecard: emoji RAG
// badges with coverage counts, headline metrics, and the open-gaps list.
// GitHub sanitizes inline styles but renders tables and emoji, so the
// "nice" surface is achieved with those alone.
func renderV03HTML(w io.Writer, h *Header, cells map[string]*Cell, jsonOut bool) int {
	r := rollV03(h, cells, time.Now())
	attrs := h.AttributesVM()
	m := computeMetrics(h, cells, 0)

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Fprintln(w, string(b))
		return 0
	}

	fmt.Fprintln(w, "<h3>fovea scorecard — "+h.System+" (spec v0.3)</h3>")
	fmt.Fprintln(w, "<table>")
	fmt.Fprintln(w, "<tr><th>attribute</th><th>actors</th><th>lifecycle</th><th>data</th><th>environment</th></tr>")
	for _, a := range attrs {
		g := r.GridV03[a]
		fmt.Fprintf(w, "<tr><td><code>%s</code></td>%s%s%s%s</tr>\n",
			a, cellHTML(g["actors"]), cellHTML(g["lifecycle"]), cellHTML(g["data"]), cellHTML(g["environment"]))
	}
	fmt.Fprintln(w, "</table>")
	fmt.Fprintln(w, "<p>🟢 assessed · 🟡 assumed/roadmap · 🔴 unassessed/missing · ⚪ justified N/A · <code>n/total</code> = authored cells</p>")
	fmt.Fprintf(w, "<p><b>unassessed</b> %.1f%% · <b>na unjustified</b> %d · <b>by-design measures</b> %.1f%% · <b>overdue roadmaps</b> %d</p>\n",
		100*m.PctUnassessed, m.NAUnjustified, 100*m.PctByDesign, m.OverdueRoadmap)
	if len(r.OpenGaps) == 0 {
		fmt.Fprintln(w, "<p>✅ <b>open gaps: none.</b> The grid is complete.</p>")
	} else {
		fmt.Fprintf(w, "<details><summary><b>open gaps: %d</b></summary><ul>\n", len(r.OpenGaps))
		for _, g := range r.OpenGaps {
			fmt.Fprintln(w, "<li><code>"+g+"</code></li>")
		}
		fmt.Fprintln(w, "</ul></details>")
	}
	return 0
}

// cellHTML renders one block as an emoji badge plus coverage count.
func cellHTML(s string) string {
	if s == "·" || s == "" {
		return "<td align=\"center\">·</td>"
	}
	if len(s) > 0 && s[0] == '-' {
		return "<td align=\"center\">⚪ <code>" + s[2:] + "</code></td>"
	}
	rag, cov := s[0], s[2:]
	switch rag {
	case 'G':
		return "<td align=\"center\">🟢 <code>" + cov + "</code></td>"
	case 'A':
		return "<td align=\"center\">🟡 <code>" + cov + "</code></td>"
	case 'R':
		return "<td align=\"center\">🔴 <code>" + cov + "</code></td>"
	default:
		return "<td align=\"center\">⚪ <code>" + cov + "</code></td>"
	}
}
