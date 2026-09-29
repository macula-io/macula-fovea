package core

import (
	"bytes"
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
	if out, err := exec.Command("git", "clone", "-q", bundle, dir).CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v: %s", err, out)
	}
	return dir
}

func input(t *testing.T, repo, observation, endorsement string) VerifyInput {
	return VerifyInput{
		Observation: vector(t, observation), Endorsement: vector(t, endorsement),
		RealmKey: vector(t, "realm_key"), RealmName: fixtureRealm, Profile: profile.PQHybrid,
		Assessment: GitRevision(repo, "fixture-kx"), NowMs: 1 << 62,
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
		{"a revision the repository does not hold", "s9_unknown_sha", "endorsement", nil, 7},
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
