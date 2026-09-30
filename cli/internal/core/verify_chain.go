package core

// verify_chain.go: `fovea verify --chain`, the continuity check of a kept set
// of one slot's observations (spec v0.5, 15-observations, "Verifying a
// chain"). Each record is first accepted by the nine steps; the chained ones
// are then partitioned into groups by their prev links and read as C1 to C7.

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/macula-io/macula-go/cbor"
	"github.com/macula-io/macula-go/identity"
	"github.com/macula-io/macula-go/profile"
)

// ChainInput is what a verifier holds beside the records: VerifyInput's
// realm, assessment and clock, and every endorsement of the observer it has,
// since a history can outlive one endorsement's window.
type ChainInput struct {
	Endorsements [][]byte
	RealmKey     []byte
	RealmName    string
	Profile      profile.Profile
	Assessment   Revision
	NowMs        int64
}

// ChainReport is what a set shows. The C fields list the seq of each record
// a finding names (C1 the missing seqs, C6 the number of restarts).
type ChainReport struct {
	Records, Accepted, Unchained int
	Refused                      []*Refusal
	FirstSeq, LastSeq            uint64
	FirstObserved, LastObserved  uint64
	C1                           []Gap
	C2, C3, C4, C5, C7           []uint64
	C6                           int
	Holes                        []string
	NotEveryResult               bool
	Continuous                   bool
	// Items are the accepted records, chained ones in seq order, then the
	// unchained; Refusals name the file each refusal came from.
	Items    []*Verified
	Refusals []FileRefusal
}

// FileRefusal is one record the chain check refused, and its file.
type FileRefusal struct {
	File    string
	Refusal *Refusal
}

// Gap is a run of missing seqs, From to To inclusive.
type Gap struct{ From, To uint64 }

// VerifyChain verifies each record and reports the set's continuity. Records
// of more than one slot are an error: a chain is one signer key and subject.
func VerifyChain(in ChainInput, records [][]byte) (ChainReport, error) {
	names := make([]string, len(records))
	return VerifyChainFiles(in, records, names)
}

// VerifyChainFiles is VerifyChain with the file each record was read from,
// named in its refusal.
func VerifyChainFiles(in ChainInput, records [][]byte, names []string) (ChainReport, error) {
	r := ChainReport{Records: len(records)}
	if err := oneSlot(records); err != nil {
		return ChainReport{}, err
	}
	seen := map[[32]byte]bool{}
	var chained, unchained []*Verified
	for i, wire := range records {
		v, refusal := verifyWithAny(in, wire)
		if refusal != nil {
			r.Refused = append(r.Refused, refusal)
			r.Refusals = append(r.Refusals, FileRefusal{File: names[i], Refusal: refusal})
			continue
		}
		v.File = names[i]
		if seen[v.WireHash] {
			continue
		}
		seen[v.WireHash] = true
		r.Accepted++
		if !v.Chained {
			r.Unchained++
			unchained = append(unchained, v)
			continue
		}
		chained = append(chained, v)
	}
	// The items are a sorted COPY: readChains below reads the records in the
	// order they came, as it always has.
	items := append([]*Verified{}, chained...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Seq != items[j].Seq {
			return items[i].Seq < items[j].Seq
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	r.Items = append(items, unchained...)
	readChains(&r, chained)
	r.Continuous = len(r.Refused) == 0 && len(chained) > 0 && !r.NotEveryResult &&
		len(r.C1)+len(r.C2)+len(r.C3)+len(r.C4)+len(r.C5)+len(r.C7)+r.C6 == 0
	return r, nil
}

// oneSlot refuses a set whose records name more than one signer key and
// subject, read before any record is verified, so a record refused on its own
// cannot hide that it belongs to another slot. A record that cannot be read
// at all is left to verification to refuse.
func oneSlot(records [][]byte) error {
	var key, subject []byte
	for _, wire := range records {
		o, err := identity.DecodeObject(wire)
		if err != nil {
			continue
		}
		tbs, err := cbor.Decode(o.TBS)
		if err != nil {
			continue
		}
		if t, _ := tbs.Get("type"); t.Kind() != cbor.KindUInt || !isObservationType(t) {
			continue
		}
		sv, _ := tbs.Get("subject")
		sub, _ := sv.AsBytes()
		if key == nil {
			key, subject = o.Key, sub
		}
		if !bytes.Equal(o.Key, key) || !bytes.Equal(sub, subject) {
			return errors.New("the records are of more than one slot (signer key and subject)")
		}
	}
	return nil
}

func isObservationType(t cbor.Value) bool {
	n, ok := t.AsInt64()
	return ok && n == int64(observationType)
}

// verifyWithAny accepts a record with the first endorsement that admits it,
// or returns the refusal that got furthest.
func verifyWithAny(in ChainInput, wire []byte) (*Verified, *Refusal) {
	var furthest *Refusal
	for _, e := range in.Endorsements {
		v, refusal := VerifyObservation(VerifyInput{Observation: wire, Endorsement: e, RealmKey: in.RealmKey,
			RealmName: in.RealmName, Profile: in.Profile, Assessment: in.Assessment, NowMs: in.NowMs})
		if refusal == nil {
			return v, nil
		}
		if furthest == nil || refusal.Step > furthest.Step {
			furthest = refusal
		}
	}
	if furthest == nil {
		furthest = refuse(3, "no endorsement given")
	}
	return nil, furthest
}

// readChains partitions the chained records into groups by their links and
// reads C1 to C7 from them.
func readChains(r *ChainReport, records []*Verified) {
	if len(records) == 0 {
		return
	}
	byHash := map[[32]byte]*Verified{}
	for _, v := range records {
		byHash[v.WireHash] = v
	}
	followers := map[[32]byte][]*Verified{}
	var heads []*Verified
	for _, v := range records {
		var prev [32]byte
		copy(prev[:], v.Prev)
		if _, linked := byHash[prev]; linked && v.Seq > 0 {
			followers[prev] = append(followers[prev], v)
			continue
		}
		heads = append(heads, v)
	}
	// Groups begun at a true head (prev zero, or a prev naming no record held)
	// are ordered and compared below. A fork's branches begin at a follower of
	// a record held; the fork is C2 where it is found, and its branches are
	// read inside, not compared again.
	var groups, branches [][]*Verified
	var walk func(group []*Verified, branch bool)
	walk = func(group []*Verified, branch bool) {
		last := group[len(group)-1]
		next := followers[last.WireHash]
		sort.Slice(next, func(i, j int) bool { return next[i].CreatedAt < next[j].CreatedAt })
		switch {
		case len(next) == 1:
			walk(append(group, next[0]), branch)
			return
		case len(next) > 1:
			r.C2 = append(r.C2, next[1].Seq)
			for _, n := range next {
				walk([]*Verified{n}, true)
			}
		}
		if branch {
			branches = append(branches, group)
		} else {
			groups = append(groups, group)
		}
	}
	for _, h := range heads {
		walk([]*Verified{h}, false)
	}
	for _, g := range append(append([][]*Verified{}, groups...), branches...) {
		readGroup(r, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0].CreatedAt < groups[j][0].CreatedAt })
	for i := 1; i < len(groups); i++ {
		before, g := groups[i-1], groups[i]
		head, highest := g[0], before[len(before)-1]
		switch {
		case head.CreatedAt <= highest.CreatedAt:
			r.C2 = append(r.C2, head.Seq)
		case head.Seq == 0:
			r.C6++
			r.Holes = append(r.Holes, fmt.Sprintf("restart after seq %d (%s), at %s", highest.Seq, ms(highest.ObservedAt), ms(head.ObservedAt)))
		case head.Seq == highest.Seq+1:
			// Nothing missing between, yet the head does not follow the record
			// held at the previous seq: its prev names another record there.
			r.C2 = append(r.C2, head.Seq)
		case head.Seq > highest.Seq:
			r.C1 = append(r.C1, Gap{highest.Seq + 1, head.Seq - 1})
		default:
			r.C2 = append(r.C2, head.Seq)
		}
	}
	if len(groups) == 0 {
		// Every record follows another: a cycle cannot be signed, so this is
		// only a set of branches; its forks are already C2.
		groups = branches
	}
	first, last := groups[0][0], groups[len(groups)-1]
	r.FirstSeq, r.FirstObserved = first.Seq, first.ObservedAt
	r.LastSeq, r.LastObserved = last[len(last)-1].Seq, last[len(last)-1].ObservedAt
}

// readGroup reads C3, C4, C5 and C7 within one group, and the publish policy
// of every revision it names.
func readGroup(r *ChainReport, g []*Verified) {
	for i, v := range g {
		everyResult := v.revision.Policy.Publish == "every_result"
		r.NotEveryResult = r.NotEveryResult || !everyResult
		if everyResult && v.CreatedAt > v.ObservedAt+clockToleranceMs {
			r.C7 = append(r.C7, v.Seq)
		}
		if i == 0 {
			continue
		}
		before := g[i-1]
		if v.Seq != before.Seq+1 {
			r.C3 = append(r.C3, v.Seq)
		}
		if v.CreatedAt <= before.CreatedAt {
			r.C4 = append(r.C4, v.Seq)
		}
		cadence := max(cadenceMs(before.revision), cadenceMs(v.revision))
		slack := min(uint64(clockToleranceMs), cadence/2)
		if absDiff(v.ObservedAt, before.ObservedAt) > cadence+slack {
			r.C5 = append(r.C5, v.Seq)
		}
	}
}

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// cadenceMs is a revision's policy.cadence in milliseconds; the lint has
// checked its form.
func cadenceMs(h *Header) uint64 {
	seconds, _ := cadenceSeconds(h.Policy.Cadence)
	return uint64(seconds) * 1000
}
