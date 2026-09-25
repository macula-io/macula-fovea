package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

// ---- GitHub REST client (stdlib only) ----

type Label struct {
	Name string `json:"name"`
}

type Issue struct {
	Number int     `json:"number"`
	State  string  `json:"state"`
	Title  string  `json:"title"`
	Body   string  `json:"body"`
	Labels []Label `json:"labels"`
}

type GH struct {
	owner, repo, token string
	hc                 *http.Client
}

func NewGH(owner, repo, token string) *GH {
	return &GH{owner: owner, repo: repo, token: token, hc: &http.Client{}}
}

func (g *GH) do(method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, "https://api.github.com"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}
	resp, err := g.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("github %s %s: %s (%s)", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out != nil && len(b) > 0 {
		return json.Unmarshal(b, out)
	}
	return nil
}

func (g *GH) ListIssues(label string) ([]Issue, error) {
	var all []Issue
	for page := 1; ; page++ {
		u := fmt.Sprintf("/repos/%s/%s/issues?labels=%s&state=all&per_page=100&page=%d",
			url.PathEscape(g.owner), url.PathEscape(g.repo), url.QueryEscape(label), page)
		var batch []Issue
		if err := g.do("GET", u, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			return all, nil
		}
	}
}

func (g *GH) CreateIssue(title, body string, labels []string) (Issue, error) {
	var out Issue
	err := g.do("POST", fmt.Sprintf("/repos/%s/%s/issues", g.owner, g.repo),
		map[string]any{"title": title, "body": body, "labels": labels}, &out)
	return out, err
}

func (g *GH) UpdateIssue(n int, body string, labels []string) (Issue, error) {
	var out Issue
	err := g.do("PATCH", fmt.Sprintf("/repos/%s/%s/issues/%d", g.owner, g.repo, n),
		map[string]any{"body": body, "labels": labels}, &out)
	return out, err
}

func (g *GH) CloseIssue(n int) (Issue, error) {
	var out Issue
	err := g.do("PATCH", fmt.Sprintf("/repos/%s/%s/issues/%d", g.owner, g.repo, n),
		map[string]any{"state": "closed"}, &out)
	return out, err
}

// ---- marker contract (the idempotency mechanism) ----

const markerLabel = "fovea"

var markerRe = regexp.MustCompile(`<!-- fovea-cell: ([a-z0-9_.-]+) -->`)

func markerFor(id string) string {
	return "<!-- fovea-cell: " + id + " -->"
}

func cellIDFromBody(body string) string {
	m := markerRe.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// ---- repo detection ----

func DetectRepo(dir string) (owner, repo string, err error) {
	out, err := exec.Command("git", "-C", dir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote get-url origin: %w", err)
	}
	s := strings.TrimSpace(string(out))
	// git@github.com:owner/repo.git | https://github.com/owner/repo[.git] | ssh://git@github.com/owner/repo.git
	s = strings.TrimSuffix(s, ".git")
	i := strings.LastIndex(s, "/")
	if i < 0 {
		return "", "", fmt.Errorf("cannot parse repo from %q", s)
	}
	repo = s[i+1:]
	rest := s[:i]
	j := strings.LastIndex(rest, ":")
	if j < 0 {
		j = strings.LastIndex(rest, "/")
	}
	if j < 0 {
		return "", "", fmt.Errorf("cannot parse owner from %q", s)
	}
	owner = rest[j+1:]
	return owner, repo, nil
}
