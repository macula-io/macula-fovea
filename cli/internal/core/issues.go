package core

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// IssuesOpts — the fovea issues surface. Phase 1 (sync) + phase 2 (--check)
// of spec/proposals/github-issues.
type IssuesOpts struct {
	Dir    string
	DryRun bool
	Check  bool
	Repo   string // "owner/repo" override; "" = detect from git remote
	Token  string // "" = $GITHUB_TOKEN
}

const (
	labelRoadmap = "fovea/roadmap"
	labelOverdue = "fovea/overdue"
)

func issueBody(c *Cell) string {
	return fmt.Sprintf("Cell: %s\nStatus: %s\nOwner: %s\nReview by: %s\n\nThreat:\n%s\n\n%s\n",
		c.ID, c.Status, c.Owner, c.ReviewBy, c.Threat.Definition, markerFor(c.ID))
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
func RunIssues(o IssuesOpts) int {
	h, _, err := LoadHeader(o.Dir)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	cells, _, _ := LoadCells(o.Dir, h)

	roadmap := map[string]*Cell{}
	for id, c := range cells {
		if c.Status == "roadmap" {
			roadmap[id] = c
		}
	}

	owner, repo := "owner", "repo"
	if o.Repo != "" {
		parts := strings.SplitN(o.Repo, "/", 2)
		if len(parts) != 2 {
			fmt.Printf("--repo must be owner/repo, got %q\n", o.Repo)
			return 1
		}
		owner, repo = parts[0], parts[1]
	} else {
		owner, repo, err = DetectRepo(o.Dir)
		if err != nil {
			fmt.Printf("detect repo: %v (use --repo owner/name)\n", err)
			return 1
		}
	}

	token := o.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	g := NewGH(owner, repo, token)
	existing := map[string]Issue{}
	listed := false
	if token != "" {
		issues, err := g.ListIssues(markerLabel)
		if err != nil {
			if o.DryRun {
				fmt.Printf("(dry-run) list issues: %v — proceeding without reconciliation\n", err)
			} else {
				fmt.Printf("list issues: %v\n", err)
				return 1
			}
		} else {
			listed = true
			for _, iss := range issues {
				if id := cellIDFromBody(iss.Body); id != "" {
					existing[id] = iss
				}
			}
		}
	}

	if o.Check {
		return issuesCheck(o.Dir, roadmap, existing, listed)
	}
	return issuesSync(o, roadmap, existing, listed, g, owner, repo)
}

// issuesCheck — the anti-theater trap: a roadmap cell whose linked issue is
// closed while the cell is unchanged fails the build.
func issuesCheck(dir string, roadmap map[string]*Cell, existing map[string]Issue, listed bool) int {
	if !listed {
		fmt.Println("--check without GITHUB_TOKEN: cannot verify issue state")
		return 1
	}
	errs := 0
	for id, c := range roadmap {
		if iss, ok := existing[id]; ok && iss.State == "closed" {
			fmt.Printf("error: roadmap cell %s has closed issue #%d — do the work and update the cell, or reopen the issue\n", id, iss.Number)
			errs++
		}
		if iss, ok := existing[id]; ok && isOverdue(c, time.Now()) && !hasLabel(iss, labelOverdue) {
			fmt.Printf("warning: roadmap cell %s is overdue (review_by %s) but issue #%d is not labelled %s\n", id, c.ReviewBy, iss.Number, labelOverdue)
		}
	}
	if errs > 0 {
		return 1
	}
	fmt.Printf("fovea issues --check: %d roadmap cells, no closed-issue traps\n", len(roadmap))
	return 0
}

func issuesSync(o IssuesOpts, roadmap map[string]*Cell, existing map[string]Issue, listed bool, g *GH, owner, repo string) int {
	today := time.Now()
	ids := make([]string, 0, len(roadmap))
	for id := range roadmap {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	opened, updated, closed, untouched := 0, 0, 0, 0

	// Cells that moved off roadmap: close their issues.
	for id, iss := range existing {
		if _, still := roadmap[id]; !still && iss.State == "open" {
			if o.DryRun {
				fmt.Printf("would close #%d (cell %s no longer roadmap)\n", iss.Number, id)
			} else {
				if _, err := g.CloseIssue(iss.Number); err != nil {
					fmt.Printf("close #%d: %v\n", iss.Number, err)
					return 1
				}
				fmt.Printf("closed #%d (cell %s no longer roadmap)\n", iss.Number, id)
			}
			closed++
		}
	}

	for _, id := range ids {
		c := roadmap[id]
		overdue := isOverdue(c, today)
		body := issueBody(c)
		labels := labelsFor(c, overdue)

		iss, exists := existing[id]
		switch {
		case !exists:
			if o.DryRun {
				fmt.Printf("would open: %s (%s)\n", id, labelsString(labels))
			} else if !listed {
				fmt.Printf("would open: %s (no token: set GITHUB_TOKEN to run for real)\n", id)
			} else {
				created, err := g.CreateIssue(issueTitle(c), body, labels)
				if err != nil {
					fmt.Printf("create %s: %v\n", id, err)
					return 1
				}
				fmt.Printf("opened #%d: %s\n", created.Number, id)
			}
			opened++
		case iss.State == "closed":
			fmt.Printf("untouched #%d (%s): closed but cell still roadmap — run: fovea issues --check\n", iss.Number, id)
			untouched++
		default:
			changed := iss.Body != body || !sameLabels(iss, labels)
			if changed {
				if o.DryRun {
					fmt.Printf("would update #%d (%s): %s\n", iss.Number, id, labelsString(labels))
				} else {
					if _, err := g.UpdateIssue(iss.Number, body, labels); err != nil {
						fmt.Printf("update #%d: %v\n", iss.Number, err)
						return 1
					}
					fmt.Printf("updated #%d (%s): %s\n", iss.Number, id, labelsString(labels))
				}
				updated++
			} else {
				untouched++
			}
		}
	}

	mode := ""
	if o.DryRun {
		mode = "dry-run"
	} else if !listed {
		mode = "offline"
	}
	fmt.Printf("fovea issues %s — %s/%s: %d opened, %d updated, %d closed, %d untouched\n",
		mode, owner, repo, opened, updated, closed, untouched)
	return 0
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
