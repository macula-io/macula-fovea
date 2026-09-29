package core

import (
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// v0.4 (spec 12-cell-schema, 14-instantiation, 16-probes): evidence on
// measures, probe declarations, targets and the publication policy.

// readsAsV03 is true for the versions whose scorecard is v0.3's: v0.4 adds no
// scorecard rule.
func readsAsV03(h *Header) bool { return h.Fovea == "0.3" || h.Fovea == "0.4" }

var evidenceKinds = map[string]bool{"doc": true, "test": true, "scenario": true, "probe": true}

// executableEvidence is what spec 12 accepts behind an assessed cell: evidence
// something runs, never a citation alone.
var executableEvidence = map[string]bool{"test": true, "scenario": true, "probe": true}

var scenarioRunners = map[string]bool{"godog": true, "whitebread": true, "cucumber": true}

// probeKind is one entry of the spec 16-probes registry.
type probeKind struct {
	targetKind string
	groups     []string // the expectation vocabulary
}

// probeRegistry is spec 16-probes: name, then version.
var probeRegistry = map[string]map[int]probeKind{
	"kx_group": {1: {
		targetKind: "macula_station",
		groups:     []string{"secp384r1_mlkem1024", "secp256r1_mlkem768", "x25519", "secp256r1", "secp384r1"},
	}},
}

var targetKinds = map[string]bool{"macula_station": true}

var publishModes = map[string]bool{"every_result": true, "state_changes": true}

var claimID = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// isoDuration is the subset of ISO 8601 durations spec 14 allows for a
// cadence: days, hours, minutes, seconds; no years, months or weeks, whose
// length depends on the calendar.
var isoDuration = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// lintV04 runs the v0.4 rules on a v0.4 assessment, and on an older one
// refuses the v0.4 fields it would otherwise silently ignore.
func lintV04(h *Header, cells map[string]*Cell) []Finding {
	if h.Fovea != "0.4" {
		return lintFieldsBeforeV04(h, cells)
	}
	var f []Finding
	f = append(f, lintTargets(h)...)
	f = append(f, lintPolicy(h)...)
	claims := map[string][]string{} // claim id -> cell ids, one per declaration
	for _, id := range sortedIDs(cells) {
		c := cells[id]
		executable := false
		for _, m := range allMeasures(c) {
			for _, e := range m.Evidence {
				f = append(f, lintEvidence(h, id, e)...)
				executable = executable || executableEvidence[e.Kind]
				if e.Kind == "probe" && claimID.MatchString(e.Claim) {
					claims[e.Claim] = append(claims[e.Claim], id)
				}
			}
		}
		if c.Status == "assessed" && !executable {
			f = append(f, errf(ruleAssessedWithoutExecutableEvidence, id, "status assessed but no measure carries test, scenario or probe evidence"))
		}
	}
	for claim, ids := range claims {
		if len(ids) > 1 {
			for _, id := range ids {
				f = append(f, errf(ruleClaimIDDuplicate, id, "claim %q is declared %d times; an observation must name one declaration", claim, len(ids)))
			}
		}
	}
	if h.Policy == nil && len(claims) > 0 {
		f = append(f, errf(rulePolicyMissing, "fovea.yaml", "probe declarations need a policy saying what an observer may publish"))
	}
	if h.Policy != nil {
		for _, s := range h.Policy.Suspended {
			if _, ok := claims[s]; !ok {
				f = append(f, errf(rulePolicySuspendedUnknownClaim, "fovea.yaml", "policy.suspended names %q, which no probe declaration claims", s))
			}
		}
	}
	return f
}

func lintFieldsBeforeV04(h *Header, cells map[string]*Cell) []Finding {
	var f []Finding
	if len(h.Targets) > 0 || h.Policy != nil {
		f = append(f, errf(ruleFieldNeedsV04, "fovea.yaml", "targets and policy are v0.4 fields; this assessment is fovea %q", h.Fovea))
	}
	for _, id := range sortedIDs(cells) {
		for _, m := range allMeasures(cells[id]) {
			if len(m.Evidence) > 0 {
				f = append(f, errf(ruleFieldNeedsV04, id, "evidence is a v0.4 field; this assessment is fovea %q", h.Fovea))
				break
			}
		}
	}
	return f
}

func lintEvidence(h *Header, id string, e Evidence) []Finding {
	var f []Finding
	switch e.Kind {
	case "doc", "test", "scenario":
		if strings.TrimSpace(e.Ref) == "" {
			f = append(f, errf(ruleEvidenceRefEmpty, id, "%s evidence without ref", e.Kind))
		}
		if e.Kind == "scenario" && !scenarioRunners[e.Runner] {
			f = append(f, errf(ruleEvidenceRunnerUnknown, id, "scenario runner %q is not godog, whitebread or cucumber", e.Runner))
		}
	case "probe":
		f = append(f, lintProbe(h, id, e)...)
	default:
		f = append(f, errf(ruleEvidenceKindUnknown, id, "evidence kind %q is not doc, test, scenario or probe", e.Kind))
	}
	return f
}

func lintProbe(h *Header, id string, e Evidence) []Finding {
	var f []Finding
	if !claimID.MatchString(e.Claim) {
		f = append(f, errf(ruleClaimIDInvalid, id, "claim id %q is not a lowercase identifier", e.Claim))
	}
	kind, known := probeRegistry[e.Probe][e.Version]
	if !known {
		return append(f, errf(ruleProbeUnknown, id, "probe %q version %d is not in the spec's registry", e.Probe, e.Version))
	}
	// A target of an unknown kind is reported once, where it is declared
	// (target_kind_unknown), not again at every probe that names it.
	if t, ok := h.Targets[e.Target]; !ok || (targetKinds[t.Kind] && t.Kind != kind.targetKind) {
		f = append(f, errf(ruleProbeTargetUndeclared, id, "probe %s observes a %s target; the header declares no such target %q", e.Probe, kind.targetKind, e.Target))
	}
	if why := expectationProblem(kind, e.Expect.Accepted, e.Expect.Refused); why != "" {
		f = append(f, errf(ruleProbeExpectationInvalid, id, "probe %s expectation: %s", e.Probe, why))
	}
	return f
}

// expectationProblem says what is wrong with an expectation, or "".
func expectationProblem(kind probeKind, accepted, refused []string) string {
	if len(accepted)+len(refused) == 0 {
		return "expects nothing"
	}
	// Spec 16: a refusal is evidence only beside an acceptance of the same
	// target in the same round, so refusals alone could never hold.
	if len(refused) > 0 && len(accepted) == 0 {
		return "refused groups need an accepted group beside them"
	}
	seen := map[string]string{}
	for _, list := range []struct {
		name   string
		groups []string
	}{{"accepted", accepted}, {"refused", refused}} {
		for _, g := range list.groups {
			if !contains(kind.groups, g) {
				return "group " + strconv.Quote(g) + " is not in the probe's vocabulary"
			}
			if prev, dup := seen[g]; dup {
				return "group " + strconv.Quote(g) + " is expected in " + prev + " and " + list.name
			}
			seen[g] = list.name
		}
	}
	return ""
}

func lintTargets(h *Header) []Finding {
	var f []Finding
	names := make([]string, 0, len(h.Targets))
	for n := range h.Targets {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t := h.Targets[n]
		if !targetKinds[t.Kind] {
			f = append(f, errf(ruleTargetKindUnknown, "fovea.yaml", "targets.%s.kind %q is not a spec target kind (macula_station)", n, t.Kind))
		}
		if len(t.Addresses) == 0 {
			f = append(f, errf(ruleTargetAddressInvalid, "fovea.yaml", "targets.%s has no addresses", n))
		}
		for _, a := range t.Addresses {
			if !ipLiteralWithPort(a) {
				f = append(f, errf(ruleTargetAddressInvalid, "fovea.yaml", "targets.%s address %q is not an IP literal and port in canonical form (e.g. 192.0.2.10:4433, [2001:db8::10]:4433)", n, a))
			}
		}
	}
	return f
}

// ipLiteralWithPort accepts an IP literal and port in its one canonical form
// (spec 14): "192.0.2.10:4433", "[2001:db8::10]:4433". It refuses names, since
// an observation that depended on DNS would observe whatever the resolver
// said, and any other spelling of an address, since observations carry the
// address as their record's subject and must all write it the same way.
func ipLiteralWithPort(a string) bool {
	host, port, err := net.SplitHostPort(a)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	p, err := strconv.Atoi(port)
	if ip == nil || err != nil || p <= 0 || p > 65535 {
		return false
	}
	return a == canonicalAddress(ip, p)
}

// canonicalAddress is spec 14's form: IPv4 dotted decimal (an IPv4-mapped
// IPv6 address included), or IPv6 in brackets compressed per RFC 5952 in
// lowercase with no zone, then the port in decimal without leading zeros.
func canonicalAddress(ip net.IP, port int) string {
	if v4 := ip.To4(); v4 != nil {
		return v4.String() + ":" + strconv.Itoa(port)
	}
	return "[" + ip.String() + "]:" + strconv.Itoa(port)
}

func lintPolicy(h *Header) []Finding {
	if h.Policy == nil {
		return nil
	}
	var f []Finding
	if !publishModes[h.Policy.Publish] {
		f = append(f, errf(rulePolicyPublishUnknown, "fovea.yaml", "policy.publish %q is not every_result or state_changes", h.Policy.Publish))
	}
	if s, ok := cadenceSeconds(h.Policy.Cadence); !ok || s < 60 {
		f = append(f, errf(rulePolicyCadenceInvalid, "fovea.yaml", "policy.cadence %q is not an ISO 8601 duration of at least a minute (e.g. PT1H)", h.Policy.Cadence))
	}
	return f
}

// cadenceSeconds parses the ISO 8601 subset isoDuration allows.
func cadenceSeconds(d string) (int64, bool) {
	m := isoDuration.FindStringSubmatch(d)
	if m == nil || d == "P" || d == "PT" {
		return 0, false
	}
	var total int64
	for i, unit := range []int64{86400, 3600, 60, 1} {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.ParseInt(m[i+1], 10, 64)
		if err != nil {
			return 0, false
		}
		total += n * unit
	}
	return total, true
}

func sortedIDs(cells map[string]*Cell) []string {
	ids := make([]string, 0, len(cells))
	for id := range cells {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
