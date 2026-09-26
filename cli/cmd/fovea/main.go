// fovea: CyberSec-as-Code CLI for the macula-fovea spec (v0.2 and v0.3).
// Single static binary; every command runs against an assessment directory.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/macula-io/macula-fovea/cli/internal/core"
)

const usage = `fovea: CyberSec-as-Code CLI (spec v0.3)

usage: fovea <command> [dir]

commands:
  init    <dir>   create missing cell skeletons from the header
  lint    <dir>   validate header + cells, enforce anti-theater rules
  score   <dir>   lint, then compute the scorecard metrics
  render  <dir>   print the scorecard as a markdown grid
  issues  <dir>   sync roadmap/overdue cells to GitHub issues

dir defaults to the current directory. flags: --json on score and render;
--format md|html|json (or --html) on render; --github on lint;
--help-issues for the issues command. Results go to stdout, diagnostics
to stderr.`

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

// knownFlag lists every flag of init, lint, score and render; takes says
// which command accepts which. A known flag on the wrong command is refused
// rather than ignored: `lint --json` must not look like it produced JSON.
var knownFlag = map[string]bool{
	"--json": true, "--github": true, "--html": true,
	"--format=md": true, "--format=html": true, "--format=json": true,
}

var takes = map[string]map[string]bool{
	"init":   {},
	"lint":   {"--github": true},
	"score":  {"--json": true},
	"render": {"--json": true, "--html": true, "--format=md": true, "--format=html": true, "--format=json": true},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole CLI minus the process: args exclude the program name.
func run(argv []string, stdout, stderr io.Writer) int {
	switch {
	case len(argv) < 1:
		fmt.Fprintln(stderr, usage)
		return 2
	case argv[0] == "-h" || argv[0] == "--help":
		// Help that was asked for is output, not an error.
		fmt.Fprintln(stdout, usage)
		return 0
	case argv[0] == "--help-issues":
		fmt.Fprintln(stdout, issuesUsage)
		return 0
	}
	dir := "."
	jsonOut := false
	githubOut := false
	htmlOut := false
	args := argv[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--format" && i+1 < len(args) {
			i++
			a = "--format=" + args[i]
		}
		if strings.HasPrefix(a, "--") && argv[0] != "issues" && knownFlag[a] && !takes[argv[0]][a] {
			fmt.Fprintf(stderr, "fovea: %s does not take %s\n\n%s\n", argv[0], a, usage)
			return 2
		}
		switch a {
		case "--json", "--format=json":
			jsonOut = true
		case "--github":
			githubOut = true
		case "--html", "--format=html":
			htmlOut = true
		case "--format=md":
			// markdown is render's default output
		default:
			if strings.HasPrefix(a, "--") && argv[0] != "issues" {
				fmt.Fprintf(stderr, "fovea: unknown flag %s\n\n%s\n", a, usage)
				return 2
			}
			dir = a
		}
	}
	switch argv[0] {
	case "init":
		return core.Init(dir, stdout, stderr)
	case "lint":
		return core.Lint(dir, githubOut, stdout, stderr)
	case "score":
		return core.Score(dir, jsonOut, stdout, stderr)
	case "render":
		return core.Render(dir, jsonOut, htmlOut, stdout, stderr)
	case "issues":
		return core.RunIssues(parseIssuesFlags(argv[1:]), stdout, stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}
