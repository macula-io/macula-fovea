package core

// verify.go: `fovea verify`, the offline verifier of a claim observation
// (spec v0.4, 15-observations, "Verification"). It accepts an observation
// only if all nine steps hold at T = the observation's created_at, and a
// refusal names the first step that failed.

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/macula-io/macula-go/cbor"
	"github.com/macula-io/macula-go/identity"
	"github.com/macula-io/macula-go/profile"
	"github.com/macula-io/macula-go/record"
)

// observationType is macula's domain record type of a claim observation,
// mcl_fovea.claim_observation.v1.
const observationType record.Type = 0x23

// clockToleranceMs: an observation created more than 5 minutes after the
// verifier's own clock is refused (step 1).
const clockToleranceMs = 5 * 60 * 1000

// The payload of spec 15: every key, and the kind of value it holds.
var payloadKeys = map[string]cbor.Kind{
	"system": cbor.KindText, "claim_id": cbor.KindText, "realm_id": cbor.KindBytes,
	"state": cbor.KindUInt, "probe": cbor.KindText, "probe_version": cbor.KindUInt,
	"target_address": cbor.KindText, "station_node": cbor.KindBytes,
	"expected": cbor.KindMap, "outcomes": cbor.KindMap, "assessment_sha": cbor.KindBytes,
	"publish": cbor.KindUInt, "observed_at": cbor.KindUInt,
}

var stateNames = []string{"holding", "broken", "unknown"}

// judgements is 16-probes' judgement of a round, by probe and version: the
// state a round's outcomes give against its expectation (0 accepted, 1
// refused, 2 inconclusive; 0 holding, 1 broken, 2 unknown).
var judgements = map[string]map[int]func(expected, outcomes map[string]uint64) uint64{
	"kx_group": {1: judgeKxGroup},
}

// judgeKxGroup is kx_group v1: broken when a group expected refused was
// accepted; holding when every group went as expected and at least one was
// accepted; unknown otherwise.
func judgeKxGroup(expected, outcomes map[string]uint64) uint64 {
	asExpected, anyAccepted := true, false
	for g, want := range expected {
		got := outcomes[g]
		if want == 1 && got == 0 {
			return 1
		}
		asExpected = asExpected && want == got
		anyAccepted = anyAccepted || got == 0
	}
	if asExpected && anyAccepted {
		return 0
	}
	return 2
}

// Revision reads the assessment at a commit id: its header and cells.
type Revision func(sha []byte) (*Header, map[string]*Cell, error)

// VerifyInput is what a verifier holds beside the observation (spec 15): the
// realm's public key as carried, its profile and name, the observer's realm
// member endorsement, the assessment revision the record names, and its own
// clock.
type VerifyInput struct {
	Observation []byte
	Endorsement []byte
	RealmKey    []byte
	RealmName   string
	Profile     profile.Profile
	Assessment  Revision
	NowMs       int64
}

// Verified is an accepted observation, as its record states it.
type Verified struct {
	System, ClaimID, TargetAddress, State string
	Observer                              [32]byte
	CreatedAt, ObservedAt                 uint64
	AssessmentSHA                         []byte
	Outcomes                              map[string]uint64
}

// Refusal names the first verification step that failed, and why.
type Refusal struct {
	Step   int
	Reason string
}

func (r *Refusal) Error() string { return fmt.Sprintf("refused at step %d: %s", r.Step, r.Reason) }

func refuse(step int, format string, args ...any) *Refusal {
	return &Refusal{Step: step, Reason: fmt.Sprintf(format, args...)}
}

// VerifyObservation runs the nine steps of spec 15 in order.
func VerifyObservation(in VerifyInput) (*Verified, *Refusal) {
	// 1. A macula record of type 0x23 at T = its created_at, not dated in the
	// verifier's future.
	created, err := createdAt(in.Observation)
	if err != nil {
		return nil, refuse(1, "not a macula record: %v", err)
	}
	if int64(created) > in.NowMs+clockToleranceMs {
		return nil, refuse(1, "created_at %s is more than 5 minutes after this clock", ms(created))
	}
	T := int64(created)
	verified, err := record.Verify(in.Observation, in.Profile, T)
	if err != nil {
		return nil, refuse(1, "the record does not verify: %v", err)
	}
	obs := verified.Record()
	if obs.Type != observationType {
		return nil, refuse(1, "type 0x%02x, not 0x23", uint8(obs.Type))
	}

	// 2. Exactly the payload of spec 15, and the subject it implies.
	p, reason := readPayload(obs)
	if reason != "" {
		return nil, refuse(2, "%s", reason)
	}

	// 3. The endorsement is a realm member endorsement, signed by the realm key.
	endorsed, err := record.Verify(in.Endorsement, in.Profile, T)
	if err != nil {
		return nil, refuse(3, "the endorsement does not verify at %s: %v", ms(created), err)
	}
	e := endorsed.Record()
	if e.Type != record.TypeRealmMemberEndorsement {
		return nil, refuse(3, "the endorsement is of type 0x%02x, not 0x05", uint8(e.Type))
	}
	if identity.KeyIDOf(e.Key, in.Profile) != identity.KeyIDOf(in.RealmKey, in.Profile) {
		return nil, refuse(3, "the endorsement is not signed by the realm key")
	}

	// 4. Of this realm, and so is the observation.
	realmID := sha256.Sum256([]byte(in.RealmName))
	endorsedRealm, _ := e.Payload.Get("realm_id")
	if b, _ := endorsedRealm.AsBytes(); !bytes.Equal(b, realmID[:]) {
		return nil, refuse(4, "the endorsement's realm_id is not SHA-256 of %q", in.RealmName)
	}
	if !bytes.Equal(p.realmID, realmID[:]) {
		return nil, refuse(4, "the observation's realm_id is not SHA-256 of %q", in.RealmName)
	}

	// 5. For the node whose identity key signed the observation.
	observer := identity.NodeIDOf(obs.Key, in.Profile)
	member, _ := e.Payload.Get("member_node")
	if b, _ := member.AsBytes(); !bytes.Equal(b, observer[:]) {
		return nil, refuse(5, "the endorsement admits another node than %x, the observation's signer", observer)
	}

	// 6. Admitting it at T, for at most 30 days.
	from, okFrom := uintField(e.Payload, "valid_from")
	until, okUntil := uintField(e.Payload, "valid_until")
	switch {
	case !okFrom || !okUntil:
		return nil, refuse(6, "the endorsement has no integer valid_from and valid_until")
	case until < from || until-from > uint64(record.MaxEndorsementWindowMs):
		return nil, refuse(6, "the endorsement's window is not at most 30 days")
	case created < from || created > until:
		return nil, refuse(6, "the endorsement admits %s to %s, not %s", ms(from), ms(until), ms(created))
	}

	// 7. An observer the named revision trusts.
	h, cells, err := in.Assessment(p.assessmentSHA)
	if err != nil {
		return nil, refuse(7, "assessment revision %x cannot be read: %v", p.assessmentSHA, err)
	}
	if h.Policy == nil || !slices.Contains(h.Policy.Observers, hex.EncodeToString(observer[:])) {
		return nil, refuse(7, "%x is not one of revision %x's policy.observers", observer, p.assessmentSHA)
	}

	// 8. The state the probe's judgement gives.
	if !sameKeys(p.expected, p.outcomes) {
		return nil, refuse(8, "outcomes do not name exactly the expected groups")
	}
	judge, known := judgements[p.probe][int(p.probeVersion)]
	if !known {
		return nil, refuse(8, "no judgement for probe %s version %d", p.probe, p.probeVersion)
	}
	if want := judge(p.expected, p.outcomes); want != p.state {
		return nil, refuse(8, "state %s, but the outcomes judge %s", stateNames[p.state], stateNames[want])
	}

	// 9. The revision's own claim, on a station it declares.
	if reason := declared(h, cells, p); reason != "" {
		return nil, refuse(9, "%s", reason)
	}
	return &Verified{
		System: p.system, ClaimID: p.claimID, TargetAddress: p.targetAddress, State: stateNames[p.state],
		Observer: observer, CreatedAt: created, ObservedAt: p.observedAt, AssessmentSHA: p.assessmentSHA,
		Outcomes: p.outcomes,
	}, nil
}

// createdAt reads a record's created_at before it is verified, since
// verification runs at that time. Nothing here is trusted: record.Verify
// checks the same bytes next.
func createdAt(wire []byte) (uint64, error) {
	if len(wire) > record.MaxRecordBytes {
		return 0, record.ErrRecordTooLarge
	}
	o, err := identity.DecodeObject(wire)
	if err != nil {
		return 0, err
	}
	tbs, err := cbor.Decode(o.TBS)
	if err != nil {
		return 0, err
	}
	v, present := tbs.Get("created_at")
	if !present || v.Kind() != cbor.KindUInt {
		return 0, errors.New("no created_at")
	}
	c, _ := v.AsInt64()
	return uint64(c), nil
}

// payload is an observation's payload once step 2 has read it.
type payload struct {
	system, claimID, probe, targetAddress    string
	realmID, stationNode, assessmentSHA      []byte
	state, probeVersion, publish, observedAt uint64
	expected, outcomes                       map[string]uint64
}

// readPayload is step 2: exactly spec 15's keys, each once and of its kind,
// in range, and the subject they imply. It returns a reason on refusal.
func readPayload(r record.Record) (payload, string) {
	entries, _ := r.Payload.AsMap()
	fields := map[string]cbor.Value{}
	for _, e := range entries {
		k, isText := e.Key.AsText()
		switch {
		case !isText:
			return payload{}, "a payload key that is not text"
		case hasKey(fields, k):
			return payload{}, fmt.Sprintf("payload key %q twice", k)
		}
		kind, known := payloadKeys[k]
		if !known {
			return payload{}, fmt.Sprintf("payload key %q is not spec 15's", k)
		}
		if e.Val.Kind() != kind {
			return payload{}, fmt.Sprintf("payload %s is not of its kind", k)
		}
		fields[k] = e.Val
	}
	for k := range payloadKeys {
		if !hasKey(fields, k) {
			return payload{}, fmt.Sprintf("payload key %q is missing", k)
		}
	}
	text := func(k string) string { s, _ := fields[k].AsText(); return s }
	raw := func(k string) []byte { b, _ := fields[k].AsBytes(); return b }
	num := func(k string) uint64 { n, _ := fields[k].AsInt64(); return uint64(n) }
	p := payload{
		system: text("system"), claimID: text("claim_id"), probe: text("probe"), targetAddress: text("target_address"),
		realmID: raw("realm_id"), stationNode: raw("station_node"), assessmentSHA: raw("assessment_sha"),
		state: num("state"), probeVersion: num("probe_version"), publish: num("publish"), observedAt: num("observed_at"),
	}
	var reason string
	if p.expected, reason = groups(fields["expected"], "expected", 1); reason != "" {
		return payload{}, reason
	}
	if p.outcomes, reason = groups(fields["outcomes"], "outcomes", 2); reason != "" {
		return payload{}, reason
	}
	switch {
	case len(p.realmID) != 32:
		return payload{}, "realm_id is not 32 bytes"
	case len(p.stationNode) != 32:
		return payload{}, "station_node is not 32 bytes"
	case len(p.assessmentSHA) != 20 && len(p.assessmentSHA) != 32:
		return payload{}, "assessment_sha is not 20 or 32 bytes"
	case p.state > 2:
		return payload{}, fmt.Sprintf("state %d is not 0, 1 or 2", p.state)
	case p.publish > 1:
		return payload{}, fmt.Sprintf("publish %d is not 0 or 1", p.publish)
	case p.observedAt > r.CreatedAt:
		return payload{}, "observed_at is after created_at"
	}
	subject := p.system + "\x00" + p.claimID + "\x00" + p.targetAddress
	if !bytes.Equal(r.Subject, []byte(subject)) {
		return payload{}, "the subject is not system 0x00 claim_id 0x00 target_address"
	}
	return p, ""
}

func hasKey(m map[string]cbor.Value, k string) bool { _, ok := m[k]; return ok }

// groups reads a map of text groups to unsigned values up to max.
func groups(v cbor.Value, name string, max uint64) (map[string]uint64, string) {
	entries, _ := v.AsMap()
	out := make(map[string]uint64, len(entries))
	for _, e := range entries {
		g, isText := e.Key.AsText()
		n, isInt := e.Val.AsInt64()
		switch {
		case !isText || e.Val.Kind() != cbor.KindUInt || !isInt:
			return nil, fmt.Sprintf("%s is not a map of text groups to unsigned values", name)
		case uint64(n) > max:
			return nil, fmt.Sprintf("%s %s is %d, over %d", name, g, n, max)
		}
		if _, twice := out[g]; twice {
			return nil, fmt.Sprintf("%s names %s twice", name, g)
		}
		out[g] = uint64(n)
	}
	return out, ""
}

// uintField is an unsigned field of a map. The zero cbor.Value is KindUInt,
// so a missing field is told apart by Get, never by its kind.
func uintField(v cbor.Value, k string) (uint64, bool) {
	f, present := v.Get(k)
	n, ok := f.AsInt64()
	return uint64(n), present && ok && f.Kind() == cbor.KindUInt
}

func sameKeys(a, b map[string]uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

// declared is step 9: the header's system; exactly one probe declaration of
// claim_id, of this probe and version, expecting exactly `expected'; and its
// target declaring target_address as station_node.
func declared(h *Header, cells map[string]*Cell, p payload) string {
	if h.System != p.system {
		return fmt.Sprintf("the revision's system is %q, not %q", h.System, p.system)
	}
	var found []Evidence
	for _, id := range sortedIDs(cells) {
		for _, m := range allMeasures(cells[id]) {
			for _, e := range m.Evidence {
				if e.Kind == "probe" && e.Claim == p.claimID {
					found = append(found, e)
				}
			}
		}
	}
	if len(found) != 1 {
		return fmt.Sprintf("the revision holds %d probe declarations of claim %s, not one", len(found), p.claimID)
	}
	d := found[0]
	if d.Probe != p.probe || uint64(d.Version) != p.probeVersion {
		return fmt.Sprintf("claim %s is declared as %s version %d, not %s version %d", p.claimID, d.Probe, d.Version, p.probe, p.probeVersion)
	}
	if _, known := probeRegistry[d.Probe][d.Version]; !known {
		return fmt.Sprintf("probe %s version %d is not in this verifier's registry", d.Probe, d.Version)
	}
	want := map[string]uint64{}
	for side, gs := range map[uint64][]string{0: d.Expect.Accepted, 1: d.Expect.Refused} {
		for _, g := range gs {
			if _, twice := want[g]; twice {
				return fmt.Sprintf("claim %s expects %s twice", p.claimID, g)
			}
			want[g] = side
		}
	}
	if !sameKeys(want, p.expected) {
		return fmt.Sprintf("expected is not claim %s's expectation", p.claimID)
	}
	for g, side := range want {
		if p.expected[g] != side {
			return fmt.Sprintf("expected is not claim %s's expectation", p.claimID)
		}
	}
	target, ok := h.Targets[d.Target]
	if !ok {
		return fmt.Sprintf("claim %s's target %q is not declared", p.claimID, d.Target)
	}
	for _, s := range target.Stations {
		if s.Address == p.targetAddress {
			if s.NodeID != hex.EncodeToString(p.stationNode) {
				return fmt.Sprintf("the revision declares %s as node %s, not %x", s.Address, s.NodeID, p.stationNode)
			}
			return ""
		}
	}
	return fmt.Sprintf("target %s does not list %s", d.Target, p.targetAddress)
}

func ms(t uint64) string { return time.UnixMilli(int64(t)).UTC().Format(time.RFC3339Nano) }

// ---- reading the inputs ----

// maxAssessmentBytes bounds what a revision may unpack to.
const maxAssessmentBytes = 16 << 20

// GitRevision reads the assessment at dir inside the git repository repo, at
// the commit a record names, from git's object store: no checkout is trusted,
// only the commit id.
func GitRevision(repo, dir string) Revision {
	return func(sha []byte) (*Header, map[string]*Cell, error) {
		commit := hex.EncodeToString(sha)
		kind, err := exec.Command("git", "-C", repo, "cat-file", "-t", commit).Output()
		if err != nil || strings.TrimSpace(string(kind)) != "commit" {
			return nil, nil, fmt.Errorf("%s holds no commit %s", repo, commit)
		}
		tarball, err := exec.Command("git", "-C", repo, "archive", "--format=tar", commit+":"+dir).Output()
		if err != nil {
			return nil, nil, fmt.Errorf("commit %s has no directory %s", commit, dir)
		}
		tmp, err := os.MkdirTemp("", "fovea-verify-")
		if err != nil {
			return nil, nil, err
		}
		defer os.RemoveAll(tmp)
		if err := untar(tarball, tmp); err != nil {
			return nil, nil, err
		}
		h, _, err := LoadHeader(tmp)
		if err != nil {
			return nil, nil, err
		}
		cells, _, _ := LoadCells(tmp, h)
		return h, cells, nil
	}
}

// untar writes a tar's regular files and directories under dir, refusing any
// other entry, any path leaving dir, and more than maxAssessmentBytes.
func untar(b []byte, dir string) error {
	r := tar.NewReader(bytes.NewReader(b))
	var total int64
	for {
		hd, err := r.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := path.Clean(hd.Name)
		if !filepath.IsLocal(name) {
			return fmt.Errorf("an archive path %q outside the assessment", hd.Name)
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		switch hd.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if total += hd.Size; total > maxAssessmentBytes {
				return errors.New("the assessment is over 16 MiB")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(r, hd.Size))
			if err != nil {
				return err
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return err
			}
		case tar.TypeXGlobalHeader:
			// git archive's commit id comment
		default:
			return fmt.Errorf("an archive entry %q that is not a file or directory", hd.Name)
		}
	}
}

// ReadBytes reads a file holding bytes as hex (whitespace around it ignored)
// or raw.
func ReadBytes(file string) ([]byte, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(b)); s != "" && isHex(s) {
		return hex.DecodeString(s)
	}
	return b, nil
}

func isHex(s string) bool {
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return len(s)%2 == 0
}

// ---- the command ----

const verifyUsage = `fovea verify [flags] <observation>

verifies a claim observation offline (spec v0.4, 15-observations): accepted
only if all nine steps hold at the observation's created_at. Files hold
bytes as hex or raw.

flags (all required):
  --realm-key f     the realm's public key as carried
  --realm name      the realm's name (its realm id is SHA-256 of it)
  --profile p       pq_hybrid or pq_pure
  --endorsement f   the observer's realm member endorsement record
  --repo dir        a git repository holding the assessment revision
  --path dir        the assessment's directory inside that repository

exit 0 accepted, 1 refused (the first failing step is named), 2 usage.`

// RunVerify is the verify command: args exclude the command name.
func RunVerify(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, verifyUsage)
		return 0
	}
	flags := map[string]string{}
	var files []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, value, hasValue := strings.Cut(a, "=")
		switch {
		case !strings.HasPrefix(a, "--"):
			files = append(files, a)
			continue
		case !hasValue && i+1 < len(args):
			i++
			value = args[i]
		case !hasValue:
			fmt.Fprintf(stderr, "fovea verify: %s needs a value\n\n%s\n", a, verifyUsage)
			return 2
		}
		switch name {
		case "--realm-key", "--realm", "--profile", "--endorsement", "--repo", "--path":
			flags[name] = value
		default:
			fmt.Fprintf(stderr, "fovea verify: unknown flag %s\n\n%s\n", name, verifyUsage)
			return 2
		}
	}
	missing := []string{}
	for _, f := range []string{"--realm-key", "--realm", "--profile", "--endorsement", "--repo", "--path"} {
		if flags[f] == "" {
			missing = append(missing, f)
		}
	}
	if len(files) != 1 || len(missing) > 0 {
		fmt.Fprintf(stderr, "fovea verify: needs one observation and %s\n\n%s\n", strings.Join(missing, " "), verifyUsage)
		return 2
	}
	p, err := profile.Parse(flags["--profile"])
	if err != nil {
		fmt.Fprintf(stderr, "fovea verify: %v\n", err)
		return 2
	}
	in := VerifyInput{RealmName: flags["--realm"], Profile: p, NowMs: time.Now().UnixMilli(),
		Assessment: GitRevision(flags["--repo"], flags["--path"])}
	for _, r := range []struct {
		file string
		into *[]byte
	}{{files[0], &in.Observation}, {flags["--endorsement"], &in.Endorsement}, {flags["--realm-key"], &in.RealmKey}} {
		if *r.into, err = ReadBytes(r.file); err != nil {
			fmt.Fprintf(stderr, "fovea verify: %v\n", err)
			return 2
		}
	}
	v, refusal := VerifyObservation(in)
	if refusal != nil {
		fmt.Fprintf(stderr, "fovea verify: %v\n", refusal)
		return 1
	}
	fmt.Fprintf(stdout, "accepted: %s\n", v.State)
	fmt.Fprintf(stdout, "  claim       %s / %s on %s\n", v.System, v.ClaimID, v.TargetAddress)
	fmt.Fprintf(stdout, "  observer    %x\n", v.Observer)
	fmt.Fprintf(stdout, "  assessment  %x\n", v.AssessmentSHA)
	fmt.Fprintf(stdout, "  observed    %s, signed %s\n", ms(v.ObservedAt), ms(v.CreatedAt))
	gs := make([]string, 0, len(v.Outcomes))
	for g := range v.Outcomes {
		gs = append(gs, g)
	}
	sort.Strings(gs)
	for _, g := range gs {
		fmt.Fprintf(stdout, "  %-20s %s\n", g, []string{"accepted", "refused", "inconclusive"}[v.Outcomes[g]])
	}
	return 0
}
