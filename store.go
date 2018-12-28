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
	// MoreStore uses the same logic as Store but do not store the
	// multisignature. It returns the (potentially new) multisignature at
	// the level, with a boolean indicating if there has been an entry update at
	// this level. It can be true if there was no multisignature previously, or
	// if the store has merged multiple multisignature together for example.
	MockStore(level byte, ms *MultiSignature) (*MultiSignature, bool)
	// Store saves the multi-signature if it is "better"
	// (implementation-dependent) than the one previously saved at the same
	// level. It returns true if the entry for this level has been updated,i.e.
	// if GetBest at the same level will return a new multi-signature.
	Store(level byte, ms *MultiSignature) (*MultiSignature, bool)
	// GetBest returns the "best" multisignature at the requested level. Best
	// should be interpreted as "containing the most individual contributions".
	Best(level byte) (*MultiSignature, bool)
	// Combined returns the best combined multi-signature possible containing
	// all levels below and up to the given level parameters. The resulting
	// bitset size is the size associated to the level+1 candidate set.
	// Can return nil if no signature stored yet.
	Combined(level byte) *MultiSignature
	/*// HighestCombined returns the best combined multi-signature possible. The*/
	//// bitset size is the size associated to the level in the sigpair, which is
	//// the maximum level signature + 1. It can return nil if there is no
	//// signature present so far.
	//Highest() *sigPair

	// FullSignature returns the best combined multi-signatures with the bitset
	// bitlength being the size of the registry
	FullSignature() *MultiSignature
}

type sigPair struct {
	origin int32
	level  byte
	ms     *MultiSignature
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
}

func newReplaceStore(part Partitioner, nbs func(int) BitSet) *replaceStore {
	return &replaceStore{
		nbs:  nbs,
		part: part,
		m:    make(map[byte]*MultiSignature),
	}
}

func (r *replaceStore) MockStore(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	r.Lock()
	defer r.Unlock()
	return r.unsafeCheck(level, ms)
}

func (r *replaceStore) Store(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	r.Lock()
	defer r.Unlock()
	n, ok := r.unsafeCheck(level, ms)
	if !ok {
		return nil, false
	}
	r.store(level, n)
	return n, true
}

func (r *replaceStore) unsafeCheck(level byte, ms *MultiSignature) (*MultiSignature, bool) {
	ms2, ok := r.m[level]
	if !ok {
		return ms, true
	}

	c1 := ms.Cardinality()
	c2 := ms2.Cardinality()
	if c1 > c2 {
		return ms, true
	}
	return ms2, false
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
	sigs := make([]*sigPair, 0, len(r.m))
	for k, ms := range r.m {
		sigs = append(sigs, &sigPair{level: k, ms: ms})
	}
	return r.part.CombineFull(sigs, r.nbs)
}

/*func (r *replaceStore) Highest() *sigPair {*/
//r.Lock()
//defer r.Unlock()
//sigs := make([]*sigPair, 0, len(r.m))
//var maxLevel byte
//for k, ms := range r.m {
//sigs = append(sigs, &sigPair{level: k, ms: ms})
//if k > maxLevel {
//maxLevel = k
//}
//}
//if maxLevel < byte(r.part.MaxLevel()) {
//maxLevel++
//}
//return r.part.Combine(sigs, int(maxLevel), r.nbs)
//}

func (r *replaceStore) Combined(level byte) *MultiSignature {
	r.Lock()
	defer r.Unlock()
	sigs := make([]*sigPair, 0, len(r.m))
	for k, ms := range r.m {
		if k > level {
			continue
		}
		sigs = append(sigs, &sigPair{level: k, ms: ms})
	}
	if level < byte(r.part.MaxLevel()) {
		level++
	}
	return r.part.Combine(sigs, int(level), r.nbs)
}

func (r *replaceStore) store(level byte, ms *MultiSignature) {
	if r.m[level] != nil && r.m[level].BitSet.Cardinality() > ms.Cardinality(){
		panic("It doesn't look like a good idea...")
	}
	r.m[level] = ms
	if level > r.highest {
		r.highest = level
	}
}

func (r *replaceStore) String() string {
	r.Lock()
	defer r.Unlock()

	var b bytes.Buffer
	b.WriteString("replaceStore table:\n")
	for lvl, ms := range r.m {
		b.WriteString(fmt.Sprintf("\tlevel %d : %s\n", lvl, ms))
	}
	return b.String()
}

func (s *sigPair) String() string {
	if s.ms == nil {
		return fmt.Sprintf("sig(lvl %d): nil", s.level)
	} else {
		return fmt.Sprintf("sig(lvl %d): %s", s.level, s.ms.String())
	}
}
