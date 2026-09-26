package core

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarkerRoundTrip(t *testing.T) {
	id := "operate.confidentiality"
	body := issueBody(&Cell{ID: id, Status: "roadmap", Owner: "a@b.c", ReviewBy: "2027-03-31",
		Threat: Threat{Definition: "retraction is broken"}}, "security/fovea/cells/"+id+".yaml")
	if got := cellIDFromBody(body); got != id {
		t.Fatalf("marker round-trip: got %q, want %q", got, id)
	}
	if cellIDFromBody("no marker here") != "" {
		t.Fatal("expected empty for unmarked body")
	}
}

func TestDetectRepo(t *testing.T) {
	for _, u := range []string{"git@github.com:macula-services/mcl-tube.git",
		"https://github.com/macula-services/mcl-tube.git",
		"https://github.com/macula-services/mcl-tube",
		"ssh://git@github.com/macula-services/mcl-tube.git"} {
		o, r, err := parseRemoteURL(u)
		if err != nil {
			t.Fatalf("%q: %v", u, err)
		}
		if o != "macula-services" || r != "mcl-tube" {
			t.Fatalf("%q parsed as %q/%q", u, o, r)
		}
	}
}

func TestLabelsOverdue(t *testing.T) {
	c := &Cell{ReviewBy: "2020-01-01"}
	if !isOverdue(c, mustParse("2026-01-01")) {
		t.Fatal("expected overdue")
	}
	if isOverdue(c, mustParse("2019-01-01")) {
		t.Fatal("expected not overdue before review_by")
	}
	ls := labelsFor(c, true)
	if ls[2] != labelOverdue {
		t.Fatal(ls)
	}
}

// fakeGitHub serves a fixed issues listing and records every write.
type fakeGitHub struct {
	listing []map[string]any
	writes  []string // "METHOD path body"
}

func (f *fakeGitHub) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			json.NewEncoder(w).Encode(f.listing)
			return
		}
		b, _ := io.ReadAll(r.Body)
		f.writes = append(f.writes, r.Method+" "+r.URL.Path+" "+string(b))
		json.NewEncoder(w).Encode(Issue{Number: 100 + len(f.writes), State: "open"})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func openIssueFor(n int, cellID string) map[string]any {
	return map[string]any{
		"number": n, "state": "open", "title": "security: " + cellID,
		"body":   markerFor(cellID) + "\nold body",
		"labels": []map[string]string{{"name": markerLabel}, {"name": labelRoadmap}},
	}
}

func runIssues(t *testing.T, dir string, o IssuesOpts, srv *httptest.Server) (int, string, string) {
	t.Helper()
	o.Dir, o.Repo, o.APIBase = dir, "o/r", srv.URL
	var out, errb bytes.Buffer
	code := RunIssues(o, &out, &errb)
	return code, out.String(), errb.String()
}

func TestSyncOpensRoadmapIssuesAndClosesStaleOnes(t *testing.T) {
	gh := &fakeGitHub{listing: []map[string]any{openIssueFor(7, "operate.integrity")}}
	code, out, errs := runIssues(t, materialize(t, "roadmap_cell_valid"), IssuesOpts{Token: "t"}, gh.serve(t))
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errs)
	}
	var posted, closed bool
	for _, w := range gh.writes {
		posted = posted || (strings.HasPrefix(w, "POST /repos/o/r/issues ") && strings.Contains(w, "fovea-cell: operate.confidentiality"))
		closed = closed || (strings.HasPrefix(w, "PATCH /repos/o/r/issues/7 ") && strings.Contains(w, `"closed"`))
	}
	if !posted || !closed || len(gh.writes) != 2 {
		t.Fatalf("writes %q: want one POST for the roadmap cell and one close of #7", gh.writes)
	}
}

// A YAML typo drops a cell from the loaded grid; acting on that grid would
// close the typo'd cell's issue as "no longer roadmap".
func TestIssuesRefusesWhenACellFailsToLoad(t *testing.T) {
	gh := &fakeGitHub{listing: []map[string]any{openIssueFor(7, "at_rest.integrity")}}
	code, _, errs := runIssues(t, materialize(t, "cell_unparseable"), IssuesOpts{Token: "t"}, gh.serve(t))
	if code != 1 || len(gh.writes) != 0 {
		t.Fatalf("exit %d, writes %q: want exit 1 and no writes", code, gh.writes)
	}
	if !strings.Contains(errs, "at_rest.integrity") {
		t.Fatalf("stderr does not name the failing cell: %q", errs)
	}
}

func TestIssuesRefusesOnHeaderFindings(t *testing.T) {
	gh := &fakeGitHub{listing: []map[string]any{openIssueFor(7, "operate.integrity")}}
	code, _, _ := runIssues(t, materialize(t, "grid_missing_column"), IssuesOpts{Token: "t"}, gh.serve(t))
	if code != 1 || len(gh.writes) != 0 {
		t.Fatalf("exit %d, writes %q: want exit 1 and no writes", code, gh.writes)
	}
}

func TestIssuesWithoutTokenIsAnErrorUnlessDryRun(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	gh := &fakeGitHub{}
	srv := gh.serve(t)
	dir := materialize(t, "roadmap_cell_valid")
	if code, out, _ := runIssues(t, dir, IssuesOpts{}, srv); code != 1 {
		t.Fatalf("no token, no --dry-run: exit %d, want 1\n%s", code, out)
	}
	if code, out, errs := runIssues(t, dir, IssuesOpts{DryRun: true}, srv); code != 0 {
		t.Fatalf("no token with --dry-run: exit %d, want 0\n%s%s", code, out, errs)
	}
	if len(gh.writes) != 0 {
		t.Fatalf("writes without a token: %q", gh.writes)
	}
}

// The issues listing endpoint returns pull requests too; one carrying a
// fovea marker must never be edited or closed.
func TestIssuesNeverPatchesPullRequests(t *testing.T) {
	pr := openIssueFor(9, "operate.integrity")
	pr["pull_request"] = map[string]string{"url": "https://api.github.com/repos/o/r/pulls/9"}
	gh := &fakeGitHub{listing: []map[string]any{pr}}
	code, out, errs := runIssues(t, materialize(t, "valid_complete"), IssuesOpts{Token: "t"}, gh.serve(t))
	if code != 0 || len(gh.writes) != 0 {
		t.Fatalf("exit %d, writes %q: want exit 0 and no writes\n%s%s", code, gh.writes, out, errs)
	}
}

func TestGitHubClientHasATimeout(t *testing.T) {
	if NewGH("o", "r", "t").hc.Timeout <= 0 {
		t.Fatal("GitHub client has no timeout: a hung API call hangs CI forever")
	}
}
