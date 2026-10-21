package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IssuesOpts is the fovea issues surface. Phase 1 (sync) + phase 2 (--check)
// of spec/proposals/github-issues.
type IssuesOpts struct {
	Dir    string
	DryRun bool
	Check  bool
	Repo   string // "owner/repo" override; "" = detect from git remote
	Token  string // "" = $GITHUB_TOKEN
	// APIBase is the GitHub REST root; "" means https://api.github.com.
	APIBase string
}

const (
	labelRoadmap = "fovea/roadmap"
	labelOverdue = "fovea/overdue"
)

func issueBody(c *Cell, cellPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Cell: %s\n", c.ID)
	if cellPath != "" {
		fmt.Fprintf(&b, "File: %s\n", cellPath)
	}
	fmt.Fprintf(&b, "Status: %s\nOwner: %s\nReview by: %s\n\nThreat:\n%s\n",
		c.Status, c.Owner, c.ReviewBy, c.Threat.Definition)
	if len(c.Threat.Manifestations) > 0 {
		b.WriteString("\nManifestations:\n")
		for _, m := range c.Threat.Manifestations {
			fmt.Fprintf(&b, "- %s\n", m)
		}
	}
	gaps := append([]Measure{}, c.Defense.Countermeasures...)
	gaps = append(gaps, c.Defense.Recovery...)
	wrote := false
	for _, m := range gaps {
		if m.Status != "roadmap" {
			continue
		}
		if !wrote {
			b.WriteString("\nThe gap (what closing this means):\n")
			wrote = true
		}
		fmt.Fprintf(&b, "- %s\n", m.Measure)
		if m.Source != "" {
			fmt.Fprintf(&b, "  source: %s\n", m.Source)
		}
	}
	if strings.TrimSpace(c.Notes) != "" {
		fmt.Fprintf(&b, "\nNotes:\n%s\n", c.Notes)
	}
	fmt.Fprintf(&b, "\n%s\n", markerFor(c.ID))
	return b.String()
}

func issueTitle(c *Cell) string {
	return "security: " + c.ID
}

func labelsFor(c *Cell, overdue bool) []string {
	ls := []string{markerLabel, labelRoadmap}
	if overdue {
		ls = append(ls, labelOverdue)
	}
	return ls
}

func isOverdue(c *Cell, today time.Time) bool {
	if c.ReviewBy == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", c.ReviewBy)
	return err == nil && t.Before(today)
}

// RunIssues executes `fovea issues`. Exit 1 on --check failures or errors.
func RunIssues(o IssuesOpts, stdout, stderr io.Writer) int {
	h, hf, err := LoadHeader(o.Dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cells, _, cf := LoadCells(o.Dir, h)
	// Refuse to act on a grid that did not load cleanly. A cell that fails
	// to parse is absent from cells, and acting would close its issue as
	// "no longer roadmap"; a wrong cells_dir would close every issue.
	if load := append(hf, cf...); len(load) > 0 {
		fmt.Fprintln(stderr, "fovea issues: refusing to act; the assessment did not load cleanly:")
		printFindings(stderr, load)
		return 1
	}

	roadmap := roadmapCells(cells)

	owner, repo, ok := resolveRepo(o, stderr)
	if !ok {
		return 1
	}
	token, ok := resolveToken(o, stderr)
	if !ok {
		return 1
	}

	g := NewGH(owner, repo, token)
	if o.APIBase != "" {
		g.base = o.APIBase
	}
	existing, listed, ok := listExisting(g, token, o.DryRun, stderr)
	if !ok {
		return 1
	}

	if o.Check {
		return issuesCheck(stdout, stderr, roadmap, existing, listed)
	}
	return issuesSync(stdout, stderr, o, h, roadmap, existing, g, owner, repo)
}

func roadmapCells(cells map[string]*Cell) map[string]*Cell {
	roadmap := map[string]*Cell{}
	for id, c := range cells {
		if c.Status == "roadmap" {
			roadmap[id] = c
		}
	}
	return roadmap
}

// resolveRepo picks the owner/repo to sync against: the --repo override, or
// the repository's own git remote.
func resolveRepo(o IssuesOpts, stderr io.Writer) (string, string, bool) {
	if o.Repo == "" {
		owner, repo, err := DetectRepo(o.Dir)
		if err != nil {
			fmt.Fprintf(stderr, "detect repo: %v (use --repo owner/name)\n", err)
			return "", "", false
		}
		return owner, repo, true
	}
	parts := strings.SplitN(o.Repo, "/", 2)
	if len(parts) != 2 {
		fmt.Fprintf(stderr, "--repo must be owner/repo, got %q\n", o.Repo)
		return "", "", false
	}
	return parts[0], parts[1], true
}

// resolveToken finds the API token. A missing CI secret must fail the job, not
// pass as a no-op; --dry-run is the one mode that runs without one.
func resolveToken(o IssuesOpts, stderr io.Writer) (string, bool) {
	token := o.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" && !o.DryRun {
		fmt.Fprintln(stderr, "fovea issues: no token (set GITHUB_TOKEN or --token); use --dry-run to compute decisions without one")
		return "", false
	}
	return token, true
}

// listExisting reads the open fovea issues so sync can reconcile against them.
// Without a token (dry-run only) the "would open" lines are computed against
// an empty tracker, and the reader is told so.
func listExisting(g *GH, token string, dryRun bool, stderr io.Writer) (map[string]Issue, bool, bool) {
	existing := map[string]Issue{}
	if token == "" {
		fmt.Fprintln(stderr, "no token: existing issues not consulted; decisions assume none exist")
		return existing, false, true
	}
	issues, err := g.ListIssues(markerLabel)
	if err != nil {
		if !dryRun {
			fmt.Fprintf(stderr, "list issues: %v\n", err)
			return existing, false, false
		}
		fmt.Fprintf(stderr, "(dry-run) list issues: %v; proceeding without reconciliation\n", err)
		return existing, false, true
	}
	for _, iss := range issues {
		if id := cellIDFromBody(iss.Body); id != "" {
			existing[id] = iss
		}
	}
	return existing, true, true
}

// issuesCheck is the anti-theater trap: a roadmap cell whose linked issue is
// closed while the cell is unchanged fails the build.
func issuesCheck(w, stderr io.Writer, roadmap map[string]*Cell, existing map[string]Issue, listed bool) int {
	if !listed {
		fmt.Fprintln(stderr, "--check without GITHUB_TOKEN: cannot verify issue state")
		return 1
	}
	errs := 0
	for id, c := range roadmap {
		if iss, ok := existing[id]; ok && iss.State == "closed" {
			fmt.Fprintf(stderr, "error: roadmap cell %s has closed issue #%d: do the work and update the cell, or reopen the issue\n", id, iss.Number)
			errs++
		}
		if iss, ok := existing[id]; ok && isOverdue(c, time.Now()) && !hasLabel(iss, labelOverdue) {
			fmt.Fprintf(stderr, "warning: roadmap cell %s is overdue (review_by %s) but issue #%d is not labelled %s\n", id, c.ReviewBy, iss.Number, labelOverdue)
		}
	}
	if errs > 0 {
		return 1
	}
	fmt.Fprintf(w, "fovea issues --check: %d roadmap cells, no closed-issue traps\n", len(roadmap))
	return 0
}

func issuesSync(w, stderr io.Writer, o IssuesOpts, h *Header, roadmap map[string]*Cell, existing map[string]Issue, g *GH, owner, repo string) int {
	today := time.Now()

	// Cells that moved off roadmap: close their issues.
	closed, ok := closeDeparted(w, stderr, o, roadmap, existing, g)
	if !ok {
		return 1
	}
	opened, updated, untouched := 0, 0, 0
	for _, id := range sortedIDs(roadmap) {
		iss, exists := existing[id]
		outcome, ok := syncCell(w, stderr, o, h, roadmap[id], iss, exists, g, today)
		if !ok {
			return 1
		}
		switch outcome {
		case "opened":
			opened++
		case "updated":
			updated++
		default:
			untouched++
		}
	}

	mode := "sync"
	if o.DryRun {
		mode = "dry-run"
	}
	fmt.Fprintf(w, "fovea issues %s, %s/%s: %d opened, %d updated, %d closed, %d untouched\n",
		mode, owner, repo, opened, updated, closed, untouched)
	return 0
}

// closeDeparted closes the issues of cells that are no longer roadmap.
func closeDeparted(w, stderr io.Writer, o IssuesOpts, roadmap map[string]*Cell, existing map[string]Issue, g *GH) (int, bool) {
	closed := 0
	for id, iss := range existing {
		if _, still := roadmap[id]; still || iss.State != "open" {
			continue
		}
		if err := closeIssue(w, stderr, o, id, iss.Number, g); err != nil {
			return closed, false
		}
		closed++
	}
	return closed, true
}

func closeIssue(w, stderr io.Writer, o IssuesOpts, id string, num int, g *GH) error {
	if o.DryRun {
		fmt.Fprintf(w, "would close #%d (cell %s no longer roadmap)\n", num, id)
		return nil
	}
	if _, err := g.CloseIssue(num); err != nil {
		fmt.Fprintf(stderr, "close #%d: %v\n", num, err)
		return err
	}
	fmt.Fprintf(w, "closed #%d (cell %s no longer roadmap)\n", num, id)
	return nil
}

// syncCell reconciles one roadmap cell with the issue that tracks it, and
// names what happened: "opened", "updated" or "untouched".
func syncCell(w, stderr io.Writer, o IssuesOpts, h *Header, c *Cell, iss Issue, exists bool, g *GH, today time.Time) (string, bool) {
	body := issueBody(c, filepath.Join(o.Dir, h.CellsDir, c.ID+".yaml"))
	labels := labelsFor(c, isOverdue(c, today))
	switch {
	case !exists:
		if err := openIssue(w, stderr, o, c, g, body, labels); err != nil {
			return "", false
		}
		return "opened", true
	case iss.State == "closed":
		fmt.Fprintf(w, "untouched #%d (%s): closed but cell still roadmap; run: fovea issues --check\n", iss.Number, c.ID)
		return "untouched", true
	case iss.Body != body || !sameLabels(iss, labels):
		if err := updateIssue(w, stderr, o, c.ID, iss.Number, g, body, labels); err != nil {
			return "", false
		}
		return "updated", true
	}
	return "untouched", true
}

func openIssue(w, stderr io.Writer, o IssuesOpts, c *Cell, g *GH, body string, labels []string) error {
	if o.DryRun {
		fmt.Fprintf(w, "would open: %s (%s)\n", c.ID, labelsString(labels))
		return nil
	}
	created, err := g.CreateIssue(issueTitle(c), body, labels)
	if err != nil {
		fmt.Fprintf(stderr, "create %s: %v\n", c.ID, err)
		return err
	}
	fmt.Fprintf(w, "opened #%d: %s\n", created.Number, c.ID)
	return nil
}

func updateIssue(w, stderr io.Writer, o IssuesOpts, id string, num int, g *GH, body string, labels []string) error {
	if o.DryRun {
		fmt.Fprintf(w, "would update #%d (%s): %s\n", num, id, labelsString(labels))
		return nil
	}
	if _, err := g.UpdateIssue(num, body, labels); err != nil {
		fmt.Fprintf(stderr, "update #%d: %v\n", num, err)
		return err
	}
	fmt.Fprintf(w, "updated #%d (%s): %s\n", num, id, labelsString(labels))
	return nil
}

func hasLabel(iss Issue, name string) bool {
	for _, l := range iss.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

func sameLabels(iss Issue, want []string) bool {
	if len(iss.Labels) != len(want) {
		return false
	}
	set := map[string]bool{}
	for _, w := range want {
		set[w] = true
	}
	for _, l := range iss.Labels {
		if !set[l.Name] {
			return false
		}
	}
	return true
}

func labelsString(ls []string) string {
	return strings.Join(ls, ",")
}
