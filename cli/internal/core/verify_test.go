package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macula-io/macula-go/profile"
)

// The vectors in testdata/verify were signed by macula 13 (generate.escript)
// with fresh, synthetic keys. Every one but obs_holding and obs_broken fails
// exactly one step of spec 15's verification; its name says which.

const fixtureRealm = "fixture.example.org"

func vector(t *testing.T, name string) []byte {
	t.Helper()
	b, err := ReadBytes(filepath.Join("testdata", "verify", name+".hex"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fixtureRepo clones the bundled assessment revision the vectors name.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "assessments")
	bundle, _ := filepath.Abs(filepath.Join("testdata", "verify", "assessment.bundle"))
	if out, err := exec.Command("git", "clone", "-q", "-b", "main", bundle, dir).CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v: %s", err, out)
	}
	return dir
}

func input(t *testing.T, repo, observation, endorsement string) VerifyInput {
	return VerifyInput{
		Observation: vector(t, observation), Endorsement: vector(t, endorsement),
		RealmKey: vector(t, "realm_key"), RealmName: fixtureRealm, Profile: profile.PQHybrid,
		Assessment: GitRevision(repo, "HEAD", "fixture-kx"), NowMs: 1 << 62,
	}
}

func TestGenuineObservationsAreAcceptedWithTheStateTheySign(t *testing.T) {
	repo := fixtureRepo(t)
	for obs, state := range map[string]string{"obs_holding": "holding", "obs_broken": "broken"} {
		v, refusal := VerifyObservation(input(t, repo, obs, "endorsement"))
		if refusal != nil {
			t.Fatalf("%s: %v", obs, refusal)
		}
		if v.State != state || v.System != "fixture-kx" || v.ClaimID != "kx_only" || v.TargetAddress != "192.0.2.10:4433" {
			t.Errorf("%s: accepted as %+v", obs, v)
		}
	}
}

// Each defect is refused at its own step, and no earlier.
func TestEachDefectIsRefusedAtTheFirstStepItFails(t *testing.T) {
	repo := fixtureRepo(t)
	cases := []struct {
		name, observation, endorsement string
		change                         func(*VerifyInput)
		step                           int
	}{
		{"a flipped signature byte", "obs_holding", "endorsement", func(in *VerifyInput) {
			in.Observation = bytes.Clone(in.Observation)
			in.Observation[len(in.Observation)-10] ^= 1
		}, 1},
		{"dated in the verifier's future", "obs_holding", "endorsement", func(in *VerifyInput) {
			c, _ := createdAt(in.Observation)
			in.NowMs = int64(c) - 6*60*1000
		}, 1},
		{"another profile", "obs_holding", "endorsement", func(in *VerifyInput) { in.Profile = profile.PQPure }, 1},
		{"not type 0x23", "s1_type", "endorsement", nil, 1},
		{"a key spec 15 does not name", "s2_extra_key", "endorsement", nil, 2},
		{"a missing key", "s2_missing_key", "endorsement", nil, 2},
		{"a state out of range", "s2_state_range", "endorsement", nil, 2},
		{"observed after it was signed", "s2_observed_after", "endorsement", nil, 2},
		{"a subject of another address", "s2_subject", "endorsement", nil, 2},
		{"system as bytes, not text", "s2_text_as_bytes", "endorsement", nil, 2},
		{"an endorsement by another key", "obs_holding", "s3_endorsement_other_signer", nil, 3},
		{"an observation in place of the endorsement", "obs_holding", "obs_holding", nil, 3},
		{"an endorsement of another realm", "obs_holding", "s4_endorsement_other_realm", nil, 4},
		{"a realm of another name", "obs_holding", "endorsement", func(in *VerifyInput) { in.RealmName = "elsewhere.example.org" }, 4},
		{"an endorsement of another node", "obs_holding", "endorsement_b", nil, 5},
		{"an endorsement expired at T", "obs_holding", "s6_endorsement_expired", nil, 6},
		{"an endorsed node the policy does not name", "obs_by_b", "endorsement_b", nil, 7},
		{"a revision the repository does not hold", "s9_unknown_sha", "endorsement", nil, 9},
		{"a state the outcomes do not give", "s8_state_wrong", "endorsement", nil, 8},
		{"outcomes of other groups", "s8_outcome_groups", "endorsement", nil, 8},
		{"an expectation that is not the claim's", "s9_expected_differs", "endorsement", nil, 9},
		{"a station node the revision does not declare", "s9_station_node", "endorsement", nil, 9},
		{"a system the revision is not", "s9_other_system", "endorsement", nil, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := input(t, repo, c.observation, c.endorsement)
			if c.change != nil {
				c.change(&in)
			}
			v, refusal := VerifyObservation(in)
			if refusal == nil {
				t.Fatalf("accepted: %+v", v)
			}
			if refusal.Step != c.step {
				t.Fatalf("refused at step %d, want %d: %v", refusal.Step, c.step, refusal)
			}
		})
	}
}

// A commit the object store holds but the ref's history does not, such as
// an unmerged pull request head, is not the assessment's revision.
func TestARevisionOutsideTheRefsHistoryIsRefused(t *testing.T) {
	repo := fixtureRepo(t)
	for _, args := range [][]string{{"checkout", "-q", "--orphan", "elsewhere"},
		{"-c", "user.name=fixture", "-c", "user.email=fixture@example.org", "commit", "-q", "--allow-empty", "-m", "unrelated"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	in := input(t, repo, "obs_holding", "endorsement")
	if _, refusal := VerifyObservation(in); refusal == nil || refusal.Step != 9 {
		t.Fatalf("got %v, want a refusal at step 9", refusal)
	}
	in.Assessment = GitRevision(repo, "main", "fixture-kx")
	if _, refusal := VerifyObservation(in); refusal != nil {
		t.Fatalf("in main's history: %v", refusal)
	}
}

func TestVerifyCommand(t *testing.T) {
	repo := fixtureRepo(t)
	file := func(n string) string { return filepath.Join("testdata", "verify", n+".hex") }
	args := func(obs, endorsement string) []string {
		return []string{"--realm-key", file("realm_key"), "--realm", fixtureRealm, "--profile=pq_hybrid",
			"--endorsement", file(endorsement), "--repo", repo, "--path", "fixture-kx", file(obs)}
	}
	var out, errOut bytes.Buffer
	if rc := RunVerify(args("obs_holding", "endorsement"), &out, &errOut); rc != 0 || !strings.HasPrefix(out.String(), "accepted: holding\n") {
		t.Fatalf("rc %d, stdout %q, stderr %q", rc, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if rc := RunVerify(args("obs_holding", "endorsement_b"), &out, &errOut); rc != 1 || !strings.Contains(errOut.String(), "refused at step 5") || out.Len() != 0 {
		t.Fatalf("rc %d, stdout %q, stderr %q", rc, out.String(), errOut.String())
	}
	if rc := RunVerify([]string{file("obs_holding")}, &out, &errOut); rc != 2 {
		t.Fatalf("no flags: rc %d, want 2", rc)
	}
}

// A v0.5 record carries its place in its slot's chain; the per-record steps
// check seq and prev agree and that it declares the header's publish policy.
func TestV05RecordsCarryTheirPlaceInTheChain(t *testing.T) {
	repo := fixtureRepo(t)
	v, refusal := VerifyObservation(input(t, repo, "obs_v05", "endorsement"))
	if refusal != nil || !v.Chained || v.Seq != 0 {
		t.Fatalf("obs_v05: %+v, %v", v, refusal)
	}
	if v, _ := VerifyObservation(input(t, repo, "obs_holding", "endorsement")); v == nil || v.Chained {
		t.Fatalf("a v0.4 record is in no chain: %+v", v)
	}
	for obs, step := range map[string]int{"s2_prev_at_seq0": 2, "s2_zero_prev_at_seq1": 2, "s9_publish_differs": 9} {
		if _, refusal := VerifyObservation(input(t, repo, obs, "endorsement")); refusal == nil || refusal.Step != step {
			t.Errorf("%s: %v, want step %d", obs, refusal, step)
		}
	}
}

func chainRecords(t *testing.T, name string) [][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("testdata", "verify", "chain", name, "*.hex"))
	if err != nil || len(files) == 0 {
		t.Fatalf("chain %s: %v", name, err)
	}
	var out [][]byte
	for _, f := range files {
		b, err := ReadBytes(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

func chainInput(t *testing.T, repo string) ChainInput {
	in := input(t, repo, "obs_holding", "endorsement")
	return ChainInput{Endorsements: [][]byte{in.Endorsement}, RealmKey: in.RealmKey, RealmName: in.RealmName,
		Profile: in.Profile, Assessment: in.Assessment, NowMs: in.NowMs}
}

// Each chain case of spec v0.5 section 15 is reported as exactly its own
// findings and no other: own is the whole report of C1 to C7.
func TestChainFindings(t *testing.T) {
	repo := fixtureRepo(t)
	type findings struct {
		c1                 []Gap
		c2, c3, c4, c5, c7 []uint64
		c6                 int
	}
	cases := map[string]findings{
		"continuous":      {},
		"duplicate":       {},
		"with_unchained":  {},
		"c1_gap":          {c1: []Gap{{2, 2}}},
		"c2_fork":         {c2: []uint64{2}},
		"c2_fork_hidden":  {c2: []uint64{3}},
		"c3_seq_skips":    {c3: []uint64{3}},
		"c4_out_of_order": {c4: []uint64{2}},
		"c5_late":         {c5: []uint64{2}},
		"c6_restart":      {c6: 1},
		"c7_signed_late":  {c7: []uint64{1}},
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := VerifyChain(chainInput(t, repo), chainRecords(t, name))
			if err != nil {
				t.Fatal(err)
			}
			got := findings{c1: r.C1, c2: r.C2, c3: r.C3, c4: r.C4, c5: r.C5, c7: r.C7, c6: r.C6}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("findings %+v, want %+v", got, want)
			}
			clean := fmt.Sprint(want) == fmt.Sprint(findings{})
			if r.Continuous != clean || len(r.Refused) != 0 {
				t.Fatalf("continuous %v, refused %v, want continuous %v", r.Continuous, r.Refused, clean)
			}
		})
	}
	r, _ := VerifyChain(chainInput(t, repo), chainRecords(t, "continuous"))
	if r.Accepted != 5 || r.FirstSeq != 0 || r.LastSeq != 4 {
		t.Fatalf("continuous: %+v", r)
	}
	if r, _ := VerifyChain(chainInput(t, repo), chainRecords(t, "duplicate")); r.Accepted != 5 || r.Records != 6 {
		t.Fatalf("a record kept twice counts once: %+v", r)
	}
	if r, _ := VerifyChain(chainInput(t, repo), chainRecords(t, "with_unchained")); r.Unchained != 1 || r.Accepted != 6 {
		t.Fatalf("a v0.4 record of the slot is unchained: %+v", r)
	}
}

func TestAChainIsOneSlot(t *testing.T) {
	if _, err := VerifyChain(chainInput(t, fixtureRepo(t)), chainRecords(t, "two_slots")); err == nil {
		t.Fatal("records of two signers taken as one chain")
	}
}

func TestVerifyChainCommand(t *testing.T) {
	repo := fixtureRepo(t)
	file := func(n string) string { return filepath.Join("testdata", "verify", n+".hex") }
	args := func(dir string) []string {
		return []string{"--realm-key", file("realm_key"), "--realm", fixtureRealm, "--profile", "pq_hybrid",
			"--endorsement", file("endorsement"), "--repo", repo, "--path", "fixture-kx",
			"--chain", filepath.Join("testdata", "verify", "chain", dir)}
	}
	var out, errOut bytes.Buffer
	if rc := RunVerify(args("continuous"), &out, &errOut); rc != 0 || !strings.Contains(out.String(), "continuous: 5 records, seq 0 to 4") {
		t.Fatalf("rc %d, stdout %q, stderr %q", rc, out.String(), errOut.String())
	}
	out.Reset()
	if rc := RunVerify(args("c1_gap"), &out, &errOut); rc != 1 || !strings.Contains(out.String(), "C1 gap: seq 2") {
		t.Fatalf("rc %d, stdout %q", rc, out.String())
	}
}

// --json: the same verdicts, machine-readable; exit codes as without it.
func TestVerifyJSONSingle(t *testing.T) {
	repo := fixtureRepo(t)
	file := func(n string) string { return filepath.Join("testdata", "verify", n+".hex") }
	args := func(obs, endorsement string) []string {
		return []string{"--realm-key", file("realm_key"), "--realm", fixtureRealm, "--profile", "pq_hybrid",
			"--endorsement", file(endorsement), "--repo", repo, "--path", "fixture-kx", "--json", file(obs)}
	}
	var out, errOut bytes.Buffer
	if rc := RunVerify(args("obs_v05", "endorsement"), &out, &errOut); rc != 0 {
		t.Fatalf("rc %d, stderr %q", rc, errOut.String())
	}
	var got struct {
		Verdict string          `json:"verdict"`
		Refused json.RawMessage `json:"refused"`
		Record  struct {
			State         string            `json:"state"`
			StateCode     int               `json:"state_code"`
			Chained       int               `json:"chained"`
			Seq           *uint64           `json:"seq"`
			Slot          string            `json:"slot"`
			SignerKeyID   string            `json:"signer_key_id"`
			Claim         string            `json:"claim"`
			TargetAddress string            `json:"target_address"`
			ObservedAtMs  uint64            `json:"observed_at_ms"`
			Outcomes      map[string]string `json:"outcomes"`
			Assessment    struct {
				Sha string `json:"sha"`
			} `json:"assessment"`
		} `json:"record"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %s", err, out.String())
	}
	r := got.Record
	if got.Verdict != "accepted" || string(got.Refused) != "null" || r.State != "holding" || r.StateCode != 0 ||
		r.Chained != 1 || r.Seq == nil || *r.Seq != 0 || len(r.Slot) != 64 || len(r.SignerKeyID) != 64 ||
		r.Claim != "kx_only" || r.TargetAddress != "192.0.2.10:4433" || r.ObservedAtMs == 0 ||
		r.Outcomes["x25519"] != "refused" || len(r.Assessment.Sha) != 40 {
		t.Fatalf("%s", out.String())
	}
	out.Reset()
	if rc := RunVerify(args("obs_v05", "endorsement_b"), &out, &errOut); rc != 1 {
		t.Fatalf("refused: rc %d", rc)
	}
	var refused struct {
		Verdict string `json:"verdict"`
		Refused struct {
			Step int `json:"step"`
		} `json:"refused"`
		Record json.RawMessage `json:"record"`
	}
	if err := json.Unmarshal(out.Bytes(), &refused); err != nil || refused.Verdict != "refused" ||
		refused.Refused.Step != 5 || string(refused.Record) != "null" {
		t.Fatalf("%v: %s", err, out.String())
	}
}

func TestVerifyJSONChain(t *testing.T) {
	repo := fixtureRepo(t)
	file := func(n string) string { return filepath.Join("testdata", "verify", n+".hex") }
	run := func(dir string) (int, map[string]any) {
		var out, errOut bytes.Buffer
		rc := RunVerify([]string{"--realm-key", file("realm_key"), "--realm", fixtureRealm, "--profile", "pq_hybrid",
			"--endorsement", file("endorsement"), "--repo", repo, "--path", "fixture-kx", "--json",
			"--chain", filepath.Join("testdata", "verify", "chain", dir)}, &out, &errOut)
		var got map[string]any
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("%s: not JSON: %v: %s %s", dir, err, out.String(), errOut.String())
		}
		return rc, got
	}
	rc, got := run("continuous")
	if rc != 0 || got["continuous"] != float64(1) || got["chained"] != float64(5) || len(got["items"].([]any)) != 5 {
		t.Fatalf("continuous: rc %d %v", rc, got)
	}
	rc, got = run("c1_gap")
	gaps := got["gaps"].([]any)
	if rc != 1 || got["continuous"] != float64(0) || len(gaps) != 1 ||
		gaps[0].(map[string]any)["from"] != float64(2) || gaps[0].(map[string]any)["to"] != float64(2) {
		t.Fatalf("c1_gap: rc %d %v", rc, got)
	}
	var seqs []float64
	for _, it := range got["items"].([]any) {
		seqs = append(seqs, it.(map[string]any)["seq"].(float64))
	}
	if fmt.Sprint(seqs) != "[0 1 3 4]" {
		t.Fatalf("items not in seq order: %v", seqs)
	}
	_, got = run("c6_restart")
	if len(got["restarts"].([]any)) != 1 {
		t.Fatalf("c6: %v", got)
	}
	_, got = run("c2_fork")
	forks := got["forks"].([]any)
	if len(forks) != 1 || len(forks[0].(map[string]any)["hashes"].([]any)) != 2 {
		t.Fatalf("c2: %v", got)
	}
}
