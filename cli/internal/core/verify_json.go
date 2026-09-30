package core

// verify_json.go: `fovea verify --json`, the same verdicts as the human
// output, machine-readable. No booleans (0 and 1), times as milliseconds and
// ISO 8601, hex lowercase. The exit code is the same as without --json.

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"path/filepath"
	"sort"
	"time"
)

// ToolVersion is the fovea version the JSON names; the command sets it.
var ToolVersion = "dev"

var outcomeNames = []string{"accepted", "refused", "inconclusive"}

type assessmentJSON struct {
	Sha  string `json:"sha"`
	Path string `json:"path"`
	Ref  string `json:"ref"`
}

type recordJSON struct {
	File          string            `json:"file"`
	Hash          string            `json:"hash"`
	Slot          string            `json:"slot"`
	SignerKeyID   string            `json:"signer_key_id"`
	ObserverNode  string            `json:"observer_node"`
	System        string            `json:"system"`
	Claim         string            `json:"claim"`
	TargetAddress string            `json:"target_address"`
	SubjectHex    string            `json:"subject_hex"`
	StationNode   string            `json:"station_node"`
	State         string            `json:"state"`
	StateCode     int               `json:"state_code"`
	Outcomes      map[string]string `json:"outcomes"`
	ObservedAtMs  uint64            `json:"observed_at_ms"`
	ObservedAt    string            `json:"observed_at"`
	CreatedAtMs   uint64            `json:"created_at_ms"`
	CreatedAt     string            `json:"created_at"`
	Chained       int               `json:"chained"`
	Seq           *uint64           `json:"seq"`
	Prev          *string           `json:"prev"`
	Publish       string            `json:"publish"`
	Assessment    assessmentJSON    `json:"assessment"`
}

type refusalJSON struct {
	Step   int    `json:"step"`
	Reason string `json:"reason"`
}

func recordOf(file string, v *Verified, path, ref string) *recordJSON {
	if v == nil {
		return nil
	}
	r := &recordJSON{
		File: file, Hash: hex.EncodeToString(v.WireHash[:]), Slot: hex.EncodeToString(v.Slot[:]),
		SignerKeyID: hex.EncodeToString(v.KeyID[:]), ObserverNode: hex.EncodeToString(v.Observer[:]),
		System: v.System, Claim: v.ClaimID, TargetAddress: v.TargetAddress,
		SubjectHex: hex.EncodeToString(v.Subject), StationNode: hex.EncodeToString(v.StationNode),
		State: v.State, StateCode: stateCode(v.State), Outcomes: map[string]string{},
		ObservedAtMs: v.ObservedAt, ObservedAt: iso(v.ObservedAt), CreatedAtMs: v.CreatedAt, CreatedAt: iso(v.CreatedAt),
		Publish:    v.Publish,
		Assessment: assessmentJSON{Sha: hex.EncodeToString(v.AssessmentSHA), Path: path, Ref: ref},
	}
	for g, o := range v.Outcomes {
		r.Outcomes[g] = outcomeNames[o]
	}
	if v.Chained {
		seq, prev := v.Seq, hex.EncodeToString(v.Prev)
		r.Chained, r.Seq, r.Prev = 1, &seq, &prev
	}
	return r
}

func stateCode(state string) int {
	for i, s := range stateNames {
		if s == state {
			return i
		}
	}
	return -1
}

func iso(ms uint64) string { return time.UnixMilli(int64(ms)).UTC().Format(time.RFC3339Nano) }

func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// writeRecordJSON prints one record's verdict; a refused record carries no
// record fields, since nothing in it has been trusted.
func writeRecordJSON(w io.Writer, file string, v *Verified, refusal *Refusal, path, ref string) int {
	out := struct {
		Version string       `json:"fovea_version"`
		Verdict string       `json:"verdict"`
		Refused *refusalJSON `json:"refused"`
		Record  *recordJSON  `json:"record"`
	}{Version: ToolVersion, Verdict: "accepted"}
	if refusal != nil {
		out.Verdict, out.Refused = "refused", &refusalJSON{Step: refusal.Step, Reason: refusal.Reason}
		writeJSON(w, out)
		return 1
	}
	out.Record = recordOf(filepath.Base(file), v, path, ref)
	writeJSON(w, out)
	return 0
}

type seqJSON struct {
	Seq uint64 `json:"seq"`
}

type gapJSON struct {
	From uint64 `json:"from"`
	To   uint64 `json:"to"`
}

type forkJSON struct {
	Seq    uint64   `json:"seq"`
	Hashes []string `json:"hashes"`
}

type restartJSON struct {
	Seq    uint64 `json:"seq"`
	Detail string `json:"detail"`
}

type fileRefusalJSON struct {
	File   string `json:"file"`
	Step   int    `json:"step"`
	Reason string `json:"reason"`
}

// writeChainJSON prints a kept set's continuity report.
func writeChainJSON(w io.Writer, r ChainReport, path, ref string) int {
	out := struct {
		Version         string            `json:"fovea_version"`
		Continuous      int               `json:"continuous"`
		Files           int               `json:"files"`
		Accepted        int               `json:"accepted"`
		RefusedCount    int               `json:"refused_count"`
		Unchained       int               `json:"unchained"`
		Chained         int               `json:"chained"`
		SeqFirst        uint64            `json:"seq_first"`
		SeqLast         uint64            `json:"seq_last"`
		ObservedFirstMs uint64            `json:"observed_first_ms"`
		ObservedFirst   string            `json:"observed_first"`
		ObservedLastMs  uint64            `json:"observed_last_ms"`
		ObservedLast    string            `json:"observed_last"`
		NotEveryResult  int               `json:"not_every_result"`
		Gaps            []gapJSON         `json:"gaps"`
		Forks           []forkJSON        `json:"forks"`
		BrokenLinks     []seqJSON         `json:"broken_links"`
		OutOfOrder      []seqJSON         `json:"out_of_order"`
		Late            []seqJSON         `json:"late"`
		Restarts        []restartJSON     `json:"restarts"`
		SignedLate      []seqJSON         `json:"signed_late"`
		Refusals        []fileRefusalJSON `json:"refusals"`
		Items           []*recordJSON     `json:"items"`
	}{
		Version: ToolVersion, Files: r.Records, Accepted: r.Accepted, RefusedCount: len(r.Refused),
		Unchained: r.Unchained, Chained: r.Accepted - r.Unchained, SeqFirst: r.FirstSeq, SeqLast: r.LastSeq,
		ObservedFirstMs: r.FirstObserved, ObservedFirst: iso(r.FirstObserved),
		ObservedLastMs: r.LastObserved, ObservedLast: iso(r.LastObserved),
		Gaps: []gapJSON{}, Forks: []forkJSON{}, BrokenLinks: seqsJSON(r.C3), OutOfOrder: seqsJSON(r.C4),
		Late: seqsJSON(r.C5), Restarts: []restartJSON{}, SignedLate: seqsJSON(r.C7),
		Refusals: []fileRefusalJSON{}, Items: []*recordJSON{},
	}
	if r.Continuous {
		out.Continuous = 1
	}
	if r.NotEveryResult {
		out.NotEveryResult = 1
	}
	for _, g := range r.C1 {
		out.Gaps = append(out.Gaps, gapJSON{From: g.From, To: g.To})
	}
	for _, seq := range dedupe(r.C2) {
		f := forkJSON{Seq: seq, Hashes: []string{}}
		for _, v := range r.Items {
			if v.Chained && v.Seq == seq {
				f.Hashes = append(f.Hashes, hex.EncodeToString(v.WireHash[:]))
			}
		}
		out.Forks = append(out.Forks, f)
	}
	for _, hole := range r.Holes {
		out.Restarts = append(out.Restarts, restartJSON{Seq: 0, Detail: hole})
	}
	for _, f := range r.Refusals {
		out.Refusals = append(out.Refusals, fileRefusalJSON{File: filepath.Base(f.File), Step: f.Refusal.Step, Reason: f.Refusal.Reason})
	}
	for _, v := range r.Items {
		out.Items = append(out.Items, recordOf(filepath.Base(v.File), v, path, ref))
	}
	writeJSON(w, out)
	if r.Continuous {
		return 0
	}
	return 1
}

func seqsJSON(seqs []uint64) []seqJSON {
	out := []seqJSON{}
	for _, s := range seqs {
		out = append(out, seqJSON{Seq: s})
	}
	return out
}

func dedupe(seqs []uint64) []uint64 {
	seen := map[uint64]bool{}
	var out []uint64
	for _, s := range seqs {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
