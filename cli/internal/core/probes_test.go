package core

import "testing"

// station_release v1 is judged by its one outcome alone (16-probes): a
// refusal breaks it, since both of its sides are authenticated.
func TestStationReleaseIsJudgedByItsOneOutcome(t *testing.T) {
	expected := map[string]uint64{"signed_release": 0}
	for outcome, state := range map[uint64]uint64{0: 0, 1: 1, 2: 2} {
		if got := judgeStationRelease(expected, map[string]uint64{"signed_release": outcome}); got != state {
			t.Errorf("outcome %d judged %s, want %s", outcome, stateNames[got], stateNames[state])
		}
	}
	if got := judgeStationRelease(expected, map[string]uint64{"x25519": 0}); got != 2 {
		t.Errorf("outcomes without signed_release judged %s, want unknown", stateNames[got])
	}
}

// A probe is declarable from the spec version that registers it, and the
// versions compare as numbers.
func TestAProbeIsRegisteredFromItsSpecVersion(t *testing.T) {
	cases := []struct {
		probe, spec string
		want        bool
	}{
		{"kx_group", "0.4", true}, {"kx_group", "0.6", true},
		{"station_release", "0.5", false}, {"station_release", "0.6", true}, {"station_release", "0.10", true},
		{"no_such_probe", "0.6", false},
	}
	for _, c := range cases {
		if _, got := registered(c.probe, 1, c.spec); got != c.want {
			t.Errorf("%s at spec %s registered %v, want %v", c.probe, c.spec, got, c.want)
		}
	}
}
