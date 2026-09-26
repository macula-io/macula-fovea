// Package core implements fovea's model, lint rules, scorecard and grid init
// for spec v0.2 and v0.3 (identical grid and cell rules; v0.3 changes only
// the scorecard). Spec-normative rules are cited where enforced.
package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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
	Columns        map[string][]string `yaml:"columns"`
	TrustAnchorReg string              `yaml:"trust_anchor_registry"`
	Landscape      string              `yaml:"landscape"`
	CellsDir       string              `yaml:"cells_dir"`
	Team           []string            `yaml:"team"`
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
	err := readYAML(filepath.Join(dir, "fovea.yaml"), h)
	if err != nil {
		return nil, nil, err
	}
	var f []Finding
	if h.Fovea == "" || !knownVersions[h.Fovea] {
		f = append(f, errf("header_version_unknown", "fovea.yaml", "fovea must be a known spec version (0.2, 0.3), got %q", h.Fovea))
	}
	if unassigned(h.Owner) {
		f = append(f, errf("header_owner_unassigned", "fovea.yaml", "owner is unassigned"))
	}
	f = append(f, lintHeaderAttributes(h)...)
	f = append(f, lintHeaderColumns(h)...)
	if h.CellsDir == "" {
		h.CellsDir = "cells"
	}
	return h, f, nil
}

// unassigned reports whether an owner field names nobody. The comparison is
// normalised: " Unassigned " claims no more than "unassigned" does.
func unassigned(owner string) bool {
	o := strings.TrimSpace(owner)
	return o == "" || strings.EqualFold(o, "unassigned")
}

// specColumns is the fixed grid of spec 10-axes: 16 columns in four
// families. A header declares exactly these, each in its own family, plus
// optional x_-prefixed extension columns.
var specColumns = map[string][]string{
	"actors":      {"internal", "external", "trusted_partner", "machine_agent"},
	"lifecycle":   {"create", "acquire", "deliver", "operate", "admin", "decommission"},
	"data":        {"at_rest", "in_motion", "in_use"},
	"environment": {"physical_natural", "socio_legal", "temporal"},
}

// extensionAttributes are the only optional attribute rows (spec
// 11-attributes); each one not enabled carries a written justification.
var extensionAttributes = []string{"possession", "utility"}

var extensionColumn = regexp.MustCompile(`^x_[a-z0-9_]+$`)

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// lintHeaderColumns enforces the fixed grid (spec 10-axes): no spec column
// dropped, moved or doubled, no invented column, no fifth family.
func lintHeaderColumns(h *Header) []Finding {
	var f []Finding
	for fam := range h.Columns {
		if _, ok := specColumns[fam]; !ok {
			f = append(f, errf("grid_family_unknown", "fovea.yaml", "columns.%s is not a column family (actors, lifecycle, data, environment)", fam))
		}
	}
	declaredIn := map[string]string{}
	for _, fam := range families {
		for _, col := range h.Columns[fam] {
			if prev, dup := declaredIn[col]; dup {
				f = append(f, errf("grid_column_duplicate", "fovea.yaml", "column %q is declared more than once (columns.%s and columns.%s)", col, prev, fam))
				continue
			}
			declaredIn[col] = fam
			if extensionColumn.MatchString(col) || contains(specColumns[fam], col) {
				continue
			}
			if home := specFamilyOf(col); home != "" {
				f = append(f, errf("grid_column_wrong_family", "fovea.yaml", "column %q belongs to columns.%s, not columns.%s", col, home, fam))
				continue
			}
			f = append(f, errf("grid_column_unknown", "fovea.yaml", "column %q in columns.%s is not a spec column; extensions need the x_ prefix", col, fam))
		}
	}
	for _, fam := range families {
		for _, col := range specColumns[fam] {
			if _, ok := declaredIn[col]; !ok {
				f = append(f, errf("grid_missing_column", "fovea.yaml", "spec column %q is missing from columns.%s; the grid is fixed", col, fam))
			}
		}
	}
	return f
}

func specFamilyOf(col string) string {
	for _, fam := range families {
		if contains(specColumns[fam], col) {
			return fam
		}
	}
	return ""
}

// lintHeaderAttributes enforces spec 11-attributes: exactly the core five
// under core, only possession and utility under enabled, and a written
// justification for each extension left disabled.
func lintHeaderAttributes(h *Header) []Finding {
	var f []Finding
	a := h.Attributes
	seen := map[string]bool{}
	for _, x := range a.Core {
		switch {
		case seen[x]:
			f = append(f, errf("grid_attribute_duplicate", "fovea.yaml", "attribute %q is listed more than once", x))
		case contains(extensionAttributes, x):
			f = append(f, errf("grid_core_attribute_invalid", "fovea.yaml", "%q is an extension, not a core attribute; list it under attributes.enabled", x))
		case !contains(coreFive, x):
			f = append(f, errf("grid_attribute_unknown", "fovea.yaml", "attribute %q in attributes.core is not a spec attribute", x))
		}
		seen[x] = true
	}
	for _, req := range coreFive {
		if !contains(a.Core, req) {
			f = append(f, errf("grid_core_attribute_missing", "fovea.yaml", "core attribute %q missing", req))
		}
	}
	for _, x := range a.Enabled {
		switch {
		case seen[x]:
			f = append(f, errf("grid_attribute_duplicate", "fovea.yaml", "attribute %q is listed more than once", x))
		case !contains(extensionAttributes, x):
			f = append(f, errf("grid_attribute_unknown", "fovea.yaml", "attribute %q in attributes.enabled is not an extension (possession, utility)", x))
		}
		seen[x] = true
	}
	for x := range a.DisabledJustifications {
		if !contains(extensionAttributes, x) {
			f = append(f, errf("grid_attribute_unknown", "fovea.yaml", "attributes.disabled_justifications.%s is not an extension (possession, utility)", x))
		}
	}
	for _, x := range extensionAttributes {
		reason := strings.TrimSpace(a.DisabledJustifications[x])
		switch {
		case contains(a.Enabled, x) && reason != "":
			f = append(f, errf("grid_extension_contradiction", "fovea.yaml", "extension %q is enabled and also justified as disabled", x))
		case !contains(a.Enabled, x) && reason == "":
			f = append(f, errf("grid_extension_unjustified", "fovea.yaml", "extension %q is disabled without a written justification in attributes.disabled_justifications", x))
		}
	}
	return f
}

// LoadCells parses every cell present in the cells dir; returns cells by id,
// plus findings for unparseable files.
func LoadCells(dir string, h *Header) (map[string]*Cell, []string, []Finding) {
	cellsDir := filepath.Join(dir, h.CellsDir)
	out := map[string]*Cell{}
	var files []string
	var f []Finding
	ents, err := os.ReadDir(cellsDir)
	if err != nil {
		// Reported against the header, where cells_dir is set and fixed, and
		// independent of the directory the assessment happens to live in.
		return out, files, append(f, errf("cells_dir_unreadable", "fovea.yaml", "cells_dir %q cannot be read: %v", h.CellsDir, err))
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
		if err := readYAML(filepath.Join(cellsDir, name), c); err != nil {
			f = append(f, errf("cell_parse", name, "parse: %v", err))
			continue
		}
		if got := name[:len(name)-5]; got != c.ID {
			f = append(f, errf("cell_id_filename_mismatch", name, "filename base %q != id %q", got, c.ID))
		}
		out[c.ID] = c
	}
	sort.Strings(files)
	return out, files, f
}

// ---- findings ----

type Finding struct {
	Rule    string // stable rule code, e.g. grid_missing_column; conformance cases name it
	Err     bool   // true=error, false=warning
	Where   string
	Message string
}

func errf(rule, where, format string, args ...any) Finding {
	return Finding{rule, true, where, fmt.Sprintf(format, args...)}
}

func printFindings(w io.Writer, fs []Finding) (errs, warns int) {
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
		fmt.Fprintf(w, "  %-5s %-34s %s\n", kind, f.Where, f.Message)
	}
	return
}
