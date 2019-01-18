package handel

import (
	"bytes"
	"fmt"
	"sync"
)

// signatureStore is a generic interface whose role is to store received valid
// multisignature, and to be able to serve the best multisignature received so
// far at a given level. Different strategies can be implemented such as keeping
// only the best one, merging two non-colluding multi-signatures etc.
// NOTE: implementation MUST be thread-safe.
type signatureStore interface {
	SigEvaluator
	// Store saves the multi-signature if it is "better"
	// (implementation-dependent) than the one previously saved at the same
	// level. It returns true if the entry for this level has been updated,i.e.
	// if GetBest at the same level will return a new multi-signature.
	// This signature must have been verified before calling this function.
	Store(level byte, ms *MultiSignature) (*MultiSignature, bool)
	// GetBest returns the "best" multisignature at the requested level. Best
	// should be interpreted as "containing the most individual contributions".
	// it returns false if there is no signature associated to that level, true
	// otherwise.
	Best(level byte) (*MultiSignature, bool)
	// Combined returns the best combined multi-signature possible containing
	// all levels below and up to the given level parameters. The resulting
	// bitset size is the size associated to the level+1 candidate set.
	// Can return nil if no signature stored yet.
	Combined(level byte) *MultiSignature

	// FullSignature returns the best combined multi-signatures with the bitset
	// bitlength being the size of the registry
	FullSignature() *MultiSignature
}

// replaceStore is a signatureStore that only stores multisignature if it
// contains more individual contributions than what's already stored.
type replaceStore struct {
	sync.Mutex
	m       map[byte]*MultiSignature
	highest byte
	// used to create empty bitset for aggregating multi-signatures
	nbs func(int) BitSet
	// used to compute bitset length for missing multi-signatures
	part Partitioner
	c    Constructor
}

func newReplaceStore(part Partitioner, nbs func(int) BitSet, c Constructor) *replaceStore {
	return &replaceStore{
		nbs:  nbs,
		part: part,
		m:    make(map[byte]*MultiSignature),
		c:    c,
	}
}

func (r *replaceStore) Store(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	r.Lock()
	defer r.Unlock()
	n, store := r.unsafeCheckMerge(level, ms)
	if !store {
		return nil, false
	}
	r.store(level, n)
	return n, true
}

func (r *replaceStore) Evaluate(sp *incomingSig) int {
	r.Lock()
	defer r.Unlock()
	score := r.unsafeEvaluate(sp.level, sp.ms)
	if score < 0 {
		panic("can't have a negative score!")
	}
	return score
}

func (r *replaceStore) unsafeEvaluate(level byte, ms *MultiSignature) int {
	ms2 := r.m[level] // The best signature we have for this level, may be nil
	toReceive := r.part.Size(int(level))
	if ms2 != nil && ms2.IsSuperSet(ms.BitSet) {
		// We have an equal or better signature already. Ignore this new one.
		if toReceive == ms2.Cardinality() || ms.Cardinality() > 1 {
			return 0
		}
		// here, we haven't completed this level. We keep the sig as it's size 1,
		//  so it can be used in some byzantine/censorship scenarios
		return int(level)
	}

	c1 := ms.Cardinality()
	if c1 <= 0 {
		panic("no sigs in this signature?")
	}

	addedSigs := 0
	existingSigs := 0
	if ms2 == nil {
		addedSigs = c1
		existingSigs = 0
	} else {
		// We need to check that we don't overlap. If we do it will be a replacement.
		merged := ms.BitSet.Or(ms2.BitSet)
		if merged.Cardinality() != ms2.Cardinality()+c1 {
			// We can't merged, it's a replace
			addedSigs = c1 - ms2.Cardinality()
		} else {
			existingSigs = ms2.BitSet.Cardinality()
			addedSigs = merged.Cardinality()
		}
	}

	if addedSigs <= 0 {
		// At this point it can't be a single signature, it would have been
		//  caught by the isSuperSet above.
		return 0
	}

	li := int(level)
	if addedSigs+existingSigs == toReceive {
		// This completes a level! That's the best options for us. We give
		//  a greater value to the first levels/
		return 1000000 - li
	}

	// It adds value, but does not complete a level. We
	//  favorize the older level but take into account the number of sigs we receive as well.
	return 30000 - li*100 + addedSigs

}


// Returns the signature to store (can be combined with the existing one or previously verified signatures) and
//  a boolean: true if the signature should replace the previous one, false if the signature should be
//  discarded
func (r *replaceStore) unsafeCheckMerge(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	ms2 := r.m[level] // The best signature we have for this level, may be nil
	if ms2 == nil {
		return ms, true
	}

	merged := ms.BitSet.Or(ms2.BitSet)
	if merged.Cardinality() != ms2.Cardinality()+ms.Cardinality() {
		if ms2.Cardinality() >= ms.Cardinality() {
			return nil, false
		} else {
			return ms, true
		}
	} else {
		sig := r.c.Signature()
		sig = sig.Combine(ms.Signature)
		sig = sig.Combine(ms2.Signature)
		return &MultiSignature{Signature: sig, BitSet: merged}, true
	}
}



func (r *replaceStore) Best(level byte) (*MultiSignature, bool) {
	r.Lock()
	defer r.Unlock()
	ms, ok := r.m[level]
	return ms, ok
}

func (r *replaceStore) FullSignature() *MultiSignature {
	r.Lock()
	defer r.Unlock()
	sigs := make([]*incomingSig, 0, len(r.m))
	for k, ms := range r.m {
		sigs = append(sigs, &incomingSig{level: k, ms: ms})
	}
	return r.part.CombineFull(sigs, r.nbs)
}

func (r *replaceStore) Combined(level byte) *MultiSignature {
	r.Lock()
	defer r.Unlock()
	sigs := make([]*incomingSig, 0, len(r.m))
	for k, ms := range r.m {
		if k > level {
			continue
		}
		sigs = append(sigs, &incomingSig{level: k, ms: ms})
	}
	if level < byte(r.part.MaxLevel()) {
		level++
	}
	return r.part.Combine(sigs, int(level), r.nbs)
}

func (r *replaceStore) store(level byte, ms *MultiSignature) {
	r.m[level] = ms
	if level > r.highest {
		r.highest = level
	}
}

func (r *replaceStore) String() string {
	full := r.FullSignature()
	r.Lock()
	defer r.Unlock()
	var b bytes.Buffer
	b.WriteString("replaceStore table:\n")
	for lvl, ms := range r.m {
		b.WriteString(fmt.Sprintf("\tlevel %d : %s\n", lvl, ms))
	}
	b.WriteString(fmt.Sprintf("\t --> full sig: %d/%d", full.Cardinality(), full.BitLength()))
	return b.String()
}

func (s *incomingSig) String() string {
	if s.ms == nil {
		return fmt.Sprintf("sig(lvl %d): <nil>", s.level)
	}
	return fmt.Sprintf("sig(lvl %d): %s", s.level, s.ms.String())
}
