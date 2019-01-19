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
	Store(sp *incomingSig) *MultiSignature
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

	// A bitset for all the individual signatures we have already received
	//  this will allow us to evict any redundant sig quickly
	indivSigsReceived BitSet

	// A bitset for all the individual signatures we have already verified
	//  this will allow us to check quickly if we can merge them
	indivSigsVerified map[byte]BitSet

	// We keep all our verified individual signatures
	individualSigs map[byte]map[int]*MultiSignature
}

func newReplaceStore(part Partitioner, nbs func(int) BitSet, c Constructor) *replaceStore {
	indivSigsVerified := make(map[byte]BitSet)
	individualSigs := make(map[byte]map[int]*MultiSignature)
	indivSigsVerified[0] = nbs(1)
	individualSigs[0] = make(map[int]*MultiSignature)
	for _, lvl := range part.Levels() {
		indivSigsVerified[byte(lvl)] = nbs(part.Size(lvl))
		individualSigs[byte(lvl)] = make(map[int]*MultiSignature)
	}

	return &replaceStore{
		nbs:               nbs,
		part:              part,
		m:                 make(map[byte]*MultiSignature),
		c:                 c,
		indivSigsVerified: indivSigsVerified,
		individualSigs:    individualSigs,
	}
}

func (r *replaceStore) Store(sp *incomingSig) *MultiSignature {
	r.Lock()
	defer r.Unlock()

	if sp.Individual() {
		r.indivSigsVerified[sp.level].Set(sp.mappedIndex, true)
		r.individualSigs[sp.level][sp.mappedIndex] = sp.ms
	}

	n, store := r.unsafeCheckMerge(sp)
	if store {
		r.store(sp.level, n)
	}
	return n
}

func (r *replaceStore) Evaluate(sp *incomingSig) int {
	r.Lock()
	defer r.Unlock()
	score := r.unsafeEvaluate(sp)
	if score < 0 {
		panic("can't have a negative score!")
	}
	return score
}

func (r *replaceStore) unsafeEvaluate(sp *incomingSig) int {
	toReceive := r.part.Size(int(sp.level))
	curBestMs := r.m[sp.level] // The best signature we have for this level, may be nil

	if curBestMs != nil && toReceive == curBestMs.Cardinality() {
		return 0 // Completed level, we won't need this signature
	}

	if sp.Individual() && r.indivSigsVerified[sp.level].Get(int(sp.mappedIndex)) {
		return 0 // We have already verified this individual signature
	}

	if curBestMs != nil && !sp.Individual() && curBestMs.IsSuperSet(sp.ms.BitSet) {
		return 0 // We have verified an equal or better signature already. Ignore this new one.
	}

	// We take into account the individual signatures already verified we could add.
	withIndiv := sp.ms.BitSet.Or(r.indivSigsVerified[sp.level])
	newTotal := 0 // The number of signatures in our new best
	addedSigs := 0 // The number of sigs we add with our new best compared to the existing one. Can be negative
	combineCt := 0 // The number of sigs in our new best that come from combining it with individual sigs

	if curBestMs == nil {
		// the best is the new multi-sig combined with the ind. sigs
		newTotal = withIndiv.Cardinality()
		addedSigs = newTotal
		combineCt = newTotal - sp.ms.BitSet.Cardinality()
	} else {
		// We need to check that the new sig and curr sig don't overlap to merge
		if sp.ms.IntersectionCardinality(curBestMs.BitSet) != 0 {
			// We can't merge, it's a replace
			newTotal = withIndiv.Cardinality()
			addedSigs = newTotal - curBestMs.Cardinality()
			combineCt = newTotal - sp.ms.BitSet.Cardinality()
		} else {
			// We can merge our current best and the new ms. We also add individual
			//  signatures that we previously verified
			finalSet :=  withIndiv.Or(curBestMs.BitSet)
			newTotal = finalSet.Cardinality()
			addedSigs = newTotal - curBestMs.BitSet.Cardinality()
			combineCt =  finalSet.Xor(curBestMs.BitSet.Or(sp.ms.BitSet)).Cardinality()
		}
	}

	if addedSigs <= 0 {
		// It doesn't add any value, we keep only the individual signatures for
		//  byzantine fault tolerance scenario but we can remove the others.
		if sp.Individual() {
			return 1
		}
		return 0
	}

	if newTotal == toReceive {
		// This completes a level! That's the best options for us. We give
		//  a greater value to the first levels/
		return 1000000 - int(sp.level) * 10 - combineCt
	}

	// It adds value, but does not complete a level. We
	//  favorize the older level but take into account the number of sigs we receive as well.
	return 100000 - int(sp.level)*100 + addedSigs * 10 - combineCt
}

// Returns the signature to store (can be combined with the existing one or previously verified signatures) and
//  a boolean: true if the signature should replace the previous one, false if the signature should be
//  discarded
func (r *replaceStore) unsafeCheckMerge(sp *incomingSig) (*MultiSignature, bool) {
	ms2 := r.m[sp.level] // The best signature we have for this level, may be nil
	if ms2 == nil {
		// If we don't have a best for this level it means we haven't verified an
		//  individual sig yet; so we can return now without checking the individual sigs.
		return sp.ms, true
	}

	best := sp.ms
	merged := sp.ms.BitSet.Or(ms2.BitSet)
	if merged.Cardinality() == ms2.Cardinality()+sp.ms.Cardinality() {
		best.BitSet = merged
		best.Signature = ms2.Signature.Combine(sp.ms.Signature)
	}

	vl := r.indivSigsVerified[sp.level]
	iS := best.And(vl).Xor(vl)
	// in iS, all bits set mean that we can complement our current best with the corresponding individual sig.

	// There are some individual sigs that we could use.
	// Let's check first that the final signature will be larger than the existing one
	if iS.Cardinality()+best.Cardinality() <= ms2.Cardinality() {
		return nil, false
	}

	// Now we can build all this
	for pos, cont := iS.NextSet(0); cont; pos, cont = iS.NextSet(+1) {
		sig, check := r.individualSigs[sp.level][pos]
		if !check {
			panic("we should have this signature in our map")
		}
		best.BitSet.Set(pos, true)
		best.Signature = sig.Combine(best.Signature)
	}

	return best, true
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
