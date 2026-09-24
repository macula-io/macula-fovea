// Package core implements fovea's model, lint rules, scorecard and grid init
// for spec v0.2. Spec-normative rules are cited where enforced.
package core

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// ---- header ----

type Header struct {
	Fovea          string `yaml:"fovea"`
	System         string `yaml:"system"`
	AssessmentDate string `yaml:"assessment_date"`
	Owner          string `yaml:"owner"`
	Description    string `yaml:"description"`
	Scope          struct {
		In  []string `yaml:"in"`
		Out []string `yaml:"out"`
	} `yaml:"scope"`
	Attributes struct {
		Core                   []string          `yaml:"core"`
		Enabled                []string          `yaml:"enabled"`
		DisabledJustifications map[string]string `yaml:"disabled_justifications"`
	} `yaml:"attributes"`
	Columns            map[string][]string `yaml:"columns"`
	TrustAnchorReg     string              `yaml:"trust_anchor_registry"`
	Landscape          string              `yaml:"landscape"`
	CellsDir           string              `yaml:"cells_dir"`
	Team               []string            `yaml:"team"`
}

// ---- cell ----

type Measure struct {
	Measure string `yaml:"measure"`
	Status  string `yaml:"status"` // by_design | roadmap | org
	Source  string `yaml:"source"`
	Notes   string `yaml:"notes"`
}

type Threat struct {
	Definition     string   `yaml:"definition"`
	Manifestations []string `yaml:"manifestations"`
}

type Cell struct {
	ID       string `yaml:"id"`
	Status   string `yaml:"status"` // unassessed|assumed|assessed|roadmap|na
	Owner    string `yaml:"owner"`
	NAReason string `yaml:"na_reason"`
	Threat   Threat `yaml:"threat"`
	Defense  struct {
		Detection       []Measure `yaml:"detection"`
		Countermeasures []Measure `yaml:"countermeasures"`
		Recovery        []Measure `yaml:"recovery"`
	} `yaml:"defense"`
	ReviewBy string `yaml:"review_by"`
	Notes    string `yaml:"notes"`
}

// ---- derived views ----

var families = []string{"actors", "lifecycle", "data", "environment"}

// knownVersions: spec versions this CLI can render and lint. Assessments
// declare their version in the header; older headers keep their exact
// reading (spec/README: versions are forked, not branched).
var knownVersions = map[string]bool{"0.2": true, "0.3": true}

var coreFive = []string{"confidentiality", "integrity", "availability", "authenticity", "accountability"}

// DeclaredColumns returns flattened declared columns (families in canonical order).
func (h *Header) DeclaredColumns() (out []string) {
	for _, f := range families {
		out = append(out, h.Columns[f]...)
	}
	return
}

// FamilyOf maps column -> family for roll-ups.
func (h *Header) FamilyOf(c string) string {
	for _, f := range families {
		for _, col := range h.Columns[f] {
			if col == c {
				return f
			}
		}
	}
	return ""
}

// Attributes returns core + enabled extensions, canonical order.
func (h *Header) AttributesVM() []string {
	out := append([]string{}, coreFive...)
	for _, e := range h.Attributes.Enabled {
		keep := true
		for _, c := range out {
			if c == e {
				keep = false
			}
		}
		if keep {
			out = append(out, e)
		}
	}
	return out
}

// ExpectedCells is the full grid: every declared column × every attribute.
func (h *Header) ExpectedCells() []string {
	var out []string
	for _, c := range h.DeclaredColumns() {
		for _, a := range h.AttributesVM() {
			out = append(out, c+"."+a)
		}
	}
	sort.Strings(out)
	return out
}

// ---- loading ----

func readYAML(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func LoadHeader(dir string) (*Header, []Finding, error) {
	h := &Header{}
	err := readYAML(dir+"/fovea.yaml", h)
	if err != nil {
		return nil, nil, err
	}
	var f []Finding
	if h.Fovea == "" || !knownVersions[h.Fovea] {
		f = append(f, errf("fovea.yaml", "fovea must be a known spec version (0.2, 0.3), got %q", h.Fovea))
	}
	if h.Owner == "unassigned" || h.Owner == "" {
		f = append(f, errf("fovea.yaml", "owner is unassigned"))
	}
	seen := map[string]bool{}
	for _, a := range h.Attributes.Core {
		seen[a] = true
	}
	for _, req := range coreFive {
		if !seen[req] {
			f = append(f, errf("fovea.yaml", "core attribute %q missing", req))
		}
	}
	for fam := range map[string]bool{"actors": true, "lifecycle": true, "data": true, "environment": true} {
		if len(h.Columns[fam]) == 0 {
			f = append(f, errf("fovea.yaml", "columns.%s must be declared and non-empty", fam))
		}
	}
	if h.CellsDir == "" {
		h.CellsDir = "cells/"
	}
	return h, f, nil
}

// LoadCells parses every cell present in the cells dir; returns cells by id,
// plus findings for unparseable files.
func LoadCells(dir string, h *Header) (map[string]*Cell, []string, []Finding) {
	cellsDir := dir + "/" + h.CellsDir
	out := map[string]*Cell{}
	var files []string
	var f []Finding
	ents, err := os.ReadDir(cellsDir)
	if err != nil {
		return out, files, append(f, errf(cellsDir, "cannot read cells dir"))
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) < 5 || name[len(name)-5:] != ".yaml" {
			continue
		}
		files = append(files, name)
		c := &Cell{}
		if err := readYAML(cellsDir+name, c); err != nil {
			f = append(f, errf(name, "parse: %v", err))
			continue
		}
		if got := name[:len(name)-5]; got != c.ID {
			f = append(f, errf(name, "filename base %q != id %q", got, c.ID))
		}
		out[c.ID] = c
	}
	sort.Strings(files)
	return out, files, f
}

// ---- findings ----

type Finding struct {
	Err     bool // true=error, false=warning
	Where   string
	Message string
}

func errf(where, format string, args ...any) Finding {
	return Finding{true, where, fmt.Sprintf(format, args...)}
}

func warnf(where, format string, args ...any) Finding {
	return Finding{false, where, fmt.Sprintf(format, args...)}
}

func printFindings(fs []Finding) (errs, warns int) {
	sort.Slice(fs, func(i, j int) bool {
		if fs[i].Where != fs[j].Where {
			return fs[i].Where < fs[j].Where
		}
		return fs[i].Message < fs[j].Message
	})
	for _, f := range fs {
		kind := "warn "
		if f.Err {
			kind = "error"
			errs++
		} else {
			warns++
		}
		fmt.Printf("  %-5s %-34s %s\n", kind, f.Where, f.Message)
	}
	return
}
