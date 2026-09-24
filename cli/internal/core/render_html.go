package core

import (
	"encoding/json"
	"fmt"
)

// renderV03HTML emits a GitHub-job-summary-native HTML scorecard: emoji RAG
// badges with coverage counts, headline metrics, and the open-gaps list.
// GitHub sanitizes inline styles but renders tables and emoji, so the
// "nice" surface is achieved with those alone.
func renderV03HTML(h *Header, cells map[string]*Cell, jsonOut bool) int {
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
				rg := cellRAG(cell)
				if rg == ragGrey {
					continue
				}
				if rk := ragRank(rg); rk > worst {
					worst = rk
				}
			}
			if !anyCol {
				row[fam] = "·"
				cov[fam] = "·"
			} else if worst == -1 {
				row[fam] = fmt.Sprintf("- %d/%d", authored, total)
				cov[fam] = fmt.Sprintf("%d/%d", authored, total)
			} else {
				row[fam] = fmt.Sprintf("%s %d/%d", ragLetter(worst), authored, total)
				cov[fam] = fmt.Sprintf("%d/%d", authored, total)
			}
		}
		r.GridV03[a] = row
		r.Coverage[a] = cov
	}
	fillRollups(&r, attrs, r.GridV03)
	r.OpenGaps = openGaps(h, cells)
	m := computeMetrics(h, cells, 0)

	if jsonOut {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Println(string(b))
		return 0
	}

	fmt.Println("<h3>fovea scorecard — " + h.System + " (spec v0.3)</h3>")
	fmt.Println("<table>")
	fmt.Println("<tr><th>attribute</th><th>actors</th><th>lifecycle</th><th>data</th><th>environment</th></tr>")
	for _, a := range attrs {
		g := r.GridV03[a]
		fmt.Printf("<tr><td><code>%s</code></td>%s%s%s%s</tr>\n",
			a, cellHTML(g["actors"]), cellHTML(g["lifecycle"]), cellHTML(g["data"]), cellHTML(g["environment"]))
	}
	fmt.Println("</table>")
	fmt.Println("<p>🟢 assessed · 🟡 assumed/roadmap · 🔴 unassessed/missing · ⚪ justified N/A · <code>n/total</code> = authored cells</p>")
	fmt.Printf("<p><b>unassessed</b> %.1f%% · <b>na unjustified</b> %d · <b>by-design measures</b> %.1f%% · <b>overdue roadmaps</b> %d</p>\n",
		100*m.PctUnassessed, m.NAUnjustified, 100*m.PctByDesign, m.OverdueRoadmap)
	if len(r.OpenGaps) == 0 {
		fmt.Println("<p>✅ <b>open gaps: none.</b> The grid is complete.</p>")
	} else {
		fmt.Printf("<details><summary><b>open gaps — %d</b></summary><ul>\n", len(r.OpenGaps))
		const capN = 40
		for i, g := range r.OpenGaps {
			if i == capN {
				fmt.Printf("<li>… and %d more</li>\n", len(r.OpenGaps)-capN)
				break
			}
			fmt.Println("<li><code>" + g + "</code></li>")
		}
		fmt.Println("</ul></details>")
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
