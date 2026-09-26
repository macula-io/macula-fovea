// fovea — CyberSec-as-Code CLI for the macula-fovea spec (v0.2).
// Single static binary; every command runs against an assessment directory.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/macula-io/macula-fovea/cli/internal/core"
)

const usage = `fovea — CyberSec-as-Code CLI (spec v0.3)

usage: fovea <command> [dir]

commands:
  init    <dir>   create missing cell skeletons from the header
  lint    <dir>   validate header + cells, enforce anti-theater rules
  score   <dir>   lint, then compute the scorecard metrics
  render  <dir>   print the scorecard as a markdown grid
  issues  <dir>   sync roadmap/overdue cells to GitHub issues

dir defaults to the current directory. flags: --json on score/render;
--help-issues for the issues command.`

const issuesUsage = `fovea issues <dir> [flags]

syncs roadmap/overdue cells to GitHub issues, idempotently, via a body
marker (<!-- fovea-cell: <id> -->). Issues without the marker are never
touched.

flags:
  --dry-run   compute decisions, perform no writes
  --check     fail if a roadmap cell has a closed linked issue (the trap)
  --repo o/r  target repo (default: detect from git remote origin)
  --token t   GitHub token (default: $GITHUB_TOKEN)`

func parseIssuesFlags(args []string) core.IssuesOpts {
	o := core.IssuesOpts{Dir: "."}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--dry-run":
			o.DryRun = true
		case a == "--check":
			o.Check = true
		case strings.HasPrefix(a, "--repo="):
			o.Repo = strings.TrimPrefix(a, "--repo=")
		case a == "--repo" && i+1 < len(args):
			o.Repo = args[i+1]
			i++
		case strings.HasPrefix(a, "--token="):
			o.Token = strings.TrimPrefix(a, "--token=")
		case a == "--token" && i+1 < len(args):
			o.Token = args[i+1]
			i++
		default:
			o.Dir = a
		}
	}
	return o
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if os.Args[1] == "--help-issues" {
		fmt.Fprintln(os.Stderr, issuesUsage)
		os.Exit(0)
	}
	dir := "."
	jsonOut := false
	githubOut := false
	htmlOut := false
	for _, a := range os.Args[2:] {
		switch a {
		case "--json":
			jsonOut = true
		case "--github":
			githubOut = true
		case "--html", "--format=html":
			htmlOut = true
		default:
			dir = a
		}
	}
	switch os.Args[1] {
	case "init":
		os.Exit(core.Init(dir, os.Stdout, os.Stderr))
	case "lint":
		os.Exit(core.Lint(dir, githubOut, os.Stdout, os.Stderr))
	case "score":
		os.Exit(core.Score(dir, jsonOut, os.Stdout, os.Stderr))
	case "render":
		os.Exit(core.Render(dir, jsonOut, htmlOut, os.Stdout, os.Stderr))
	case "issues":
		os.Exit(core.RunIssues(parseIssuesFlags(os.Args[2:]), os.Stdout, os.Stderr))
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}
