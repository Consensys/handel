package handel

import (
	cryptoRand "crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	mathRand "math/rand"
)

// Partitioner is a generic interface holding the logic used to partition the
// nodes in different buckets.  The only Partitioner implemented is
// binTreePartition using binomial tree to partition, as in the original San
// Fermin paper.
type Partitioner interface {
	// returns the maximum number of levels this partitioning strategy will use
	// given the list of participants
	MaxLevel() int
	// Returns the size of the set of Identity at this level or an error if
	// level invalid.
	Size(level int) (int, error)
	// IdentitiesAt returns the list of Identity that composes the whole level in
	// this partition scheme.
	IdentitiesAt(level int) ([]Identity, error)
	// PickNextAt returns up to *count* Identity situated at this level. If all
	// identities have been picked already, or if no identities are found at
	// this level, it returns false.
	// TODO: it should not return false but go back to the beginning of the list
	// TODO: it should return false only if all these nodes have been sent the same signature
	PickNextAt(level, count int) ([]Identity, bool)
	// Combine takes a list of signature paired with their level and returns all
	// signatures correctly combined according to the partition strategy.
	// All signatures must be valid signatures. The return value can be nil if no
	// sigPairs have been given.It returns a MultiSignature whose's BitSet's
	// size is equal to the size of the level given in parameter + 1. The +1 is
	// there because it is a combined signature, therefore, encompassing all
	// signatures of levels up to the given level included.
	Combine(sigs []*sigPair, level int, nbs func(int) BitSet) *MultiSignature
	// CombineFull is similar to Combine but it returns the full multisignature
	// whose bitset's length is equal to the size of the registry. This length
	// corresponds to the MaxLevel() + 1 - but this level is not considered a
	// "valid" level from a Handel perspective.
	CombineFull(sigs []*sigPair, nbs func(int) BitSet) *MultiSignature
}

// binomialPartitioner is a partitioner implementation using a binomial tree
// splitting based on the common length prefix, as in the San Fermin paper.
// It returns new nodes just based on the index alone (no considerations of
// close proximity for example).
type binomialPartitioner struct {
	// candidatetree computes according to the point of view of this node's id.
	id      int
	bitsize int
	size    int
	reg     Registry
	// mapping for each level of the index of the last node picked for this
	// level
	picked map[int]int
}

// NewBinPartitioner returns a binTreePartition using the given ID as its
// anchor point in the ID list, and the given registry.
func NewBinPartitioner(id int32, reg Registry) Partitioner {
	return &binomialPartitioner{
		size:    reg.Size(),
		reg:     reg,
		id:      int(id),
		bitsize: log2(reg.Size()),
		picked:  make(map[int]int),
	}
}

func (c *binomialPartitioner) MaxLevel() int {
	return log2(c.reg.Size())
}

// IdentitiesAt returns the set of identities that corresponds to the given
// level. It uses the same logic as rangeLevel but returns directly the set of
// identities.
func (c *binomialPartitioner) IdentitiesAt(level int) ([]Identity, error) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return nil, err
	}

	ids, ok := c.reg.Identities(min, max)
	if !ok {
		return nil, errors.New("handel: registry can't find ids in range")
	}
	return ids, nil

}

// errEmptyLevel is returned when a range for a requested level is empty. This
// can happen is the number of nodes is not a perfect power of two.
var errEmptyLevel = errors.New("empty level")

// rangeLevel returns the range [min,max[ that maps to the set of identity
// comprised at the given level from the point of view of the ID of the
// binTreePartition. At each increasing level, a node should contact nodes from
// a exponentially increasing larger set of nodes, using the binomial tree
// construction as described in the San Fermin paper. Level starts at 0 (same
// node) and ends at the bitsize length + 1 (whole ID range).
// It returns errEmptyLevel if the range corresponding to the given level is
// empty.It returns an error if the level requested is out of bound.
func (c *binomialPartitioner) rangeLevel(level int) (min int, max int, err error) {
	if level < 0 || level > c.bitsize+1 {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = pow2(log2(c.size))
	var inverseIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= inverseIdx && idx >= 0 && min < max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		//fmt.Printf("id %d, idx %d, inverseIdx %d, bitsize %d, min %d, middle %d, max %d\n", c.id, idx, inverseIdx, c.bitsize, min, middle, max)

		if isSet(uint(c.id), uint(idx)) {
			// we inverse the order at the given CPL to get the candidate set.
			// Otherwise we would get the same set as c.id is in (as in
			// rangeLevelInverse)
			if idx == inverseIdx {
				max = middle
			} else {
				min = middle
			}
		} else {
			// same inversion here
			if idx == inverseIdx {
				min = middle
			} else {
				max = middle
			}
		}

	}

	//  >= because the minimum index is inclusive
	if min >= c.size {
		return 0, 0, errEmptyLevel
	}

	// > because the maximum index is exclusive
	if max > c.size {
		max = c.size
	}

	return min, max, nil
}

// rangeLevelInverse is similar to rangeLevel except that it computes the
// "opposite" group of what rangeLevel returns. It is typically needed to
// compute in what candidate set an ID belongs, or where does a signature in our
// candidate set fits. see CombineF function for one usage. It returns an error
// if the given level is out of bound.
func (c *binomialPartitioner) rangeLevelInverse(level int) (min int, max int, err error) {
	if level < 0 || level > c.bitsize+1 {
		return 0, 0, errors.New("handel: invalid level for computing candidate set")
	}

	max = pow2(log2(c.size))
	var maxIdx = level - 1
	// Use a binary-search like algo over the bitstring of the id from highest
	// bit to lower bits as long as we are above the requested common prefix
	// length to pinpoint the requested range.
	for idx := c.bitsize - 1; idx >= maxIdx && idx >= 0 && min < max; idx-- {
		middle := int(math.Floor(float64(max+min) / 2))
		//fmt.Printf("id %d, idx %d, inverseIdx %d, bitsize %d, min %d, middle %d, max %d\n", c.id, idx, maxIdx, c.bitsize, min, middle, max)

		if isSet(uint(c.id), uint(idx)) {
			min = middle
		} else {
			max = middle
		}
	}

	if max > c.size {
		max = c.size
	}
	return min, max, nil

}

// PickNext returns a set of un-picked identities at the given level, up to
// *count* elements. If no identities could have been picked, it returns false.
func (c *binomialPartitioner) PickNextAt2(level, count int) ([]Identity, bool) {
	lmin, lmax, err := c.rangeLevel(level)
	if err != nil {
		panic(err)
	}

	minPicked, ok := c.picked[level]
	if !ok {
		minPicked = lmin
	}
	if minPicked >= lmax {
		return nil, false
	}

	last := min(minPicked + count, lmax)

	ids, ok := c.reg.Identities(minPicked, last)
	if !ok {
		panic("No identifies left")
	}

	c.picked[level] = last
	return ids, true
}


// PickNext returns a set of un-picked identities at the given level, up to
// *count* elements. If no identities could have been picked, it returns false.
func (c *binomialPartitioner) PickNextAt(level, count int) ([]Identity, bool) {
	if level <= 0 {
		panic("Wrong level number")
	}


	min, max, err := c.rangeLevel(level)
	if err != nil {
		return nil, false
	}

	minPicked, ok := c.picked[level]
	if !ok {
		minPicked = min
	}

	length := max - minPicked
	if length > count {
		max = minPicked + count
	}

	ids, ok := c.reg.Identities(minPicked, max)
	if !ok || length == 0 {
		return nil, false
	}

	c.picked[level] = max
	return ids, true
}



func (c *binomialPartitioner) Size(level int) (int, error) {
	min, max, err := c.rangeLevel(level)
	if err != nil {
		return 0, err
	}
	return max - min, nil
}

func (c *binomialPartitioner) Combine(sigs []*sigPair, level int, nbs func(int) BitSet) *MultiSignature {
	if len(sigs) == 0 {
		return nil
	}

	for _, s := range sigs {
		if int(s.level) > level {
			logf("invalid combination of signature / requested level")
			return nil
		}
	}

	// taking the "rangeInverse" gives us the range covering all signatures
	// with a level inferior than "level" - it's the range nodes at the
	// corresponding candidate set expect to receive.
	globalMin, globalMax, err := c.rangeLevelInverse(level)
	if err != nil {
		logf(err.Error())
		return nil
	}
	size := globalMax - globalMin
	bitset := nbs(size)
	//fmt.Printf("\t -- Combine(lvl %d) => min %d max %d -> size %d\n", level, globalMin, globalMax, size)
	combined := func(s *sigPair, final BitSet) {
		// compute the offset of this signature compared to the global bitset
		// index
		min, _, _ := c.rangeLevel(int(s.level))
		offset := min - globalMin
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			final.Set(offset+i, bs.Get(i))
		}
	}

	return c.combineSize(sigs, bitset, combined)
}

func (c *binomialPartitioner) CombineFull(sigs []*sigPair, nbs func(int) BitSet) *MultiSignature {
	if len(sigs) == 0 {
		return nil
	}
	var finalBitSet = nbs(c.reg.Size())

	// set the bits corresponding to the level to the final bitset
	var combineBitSet = func(s *sigPair, final BitSet) {
		min, _, _ := c.rangeLevel(int(s.level))
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			final.Set(min+i, bs.Get(i))
		}
	}
	return c.combineSize(sigs, finalBitSet, combineBitSet)
}

// combineSize combines all given signature witht he combine function on the
// bitset using `bs`
func (c *binomialPartitioner) combineSize(sigs []*sigPair, bs BitSet, combine func(*sigPair, BitSet)) *MultiSignature {

	var finalSig = sigs[0].ms.Signature
	combine(sigs[0], bs)

	for _, s := range sigs[1:] {
		// combine both signatures
		finalSig = finalSig.Combine(s.ms.Signature)
		combine(s, bs)
	}
	return &MultiSignature{
		BitSet:    bs,
		Signature: finalSig,
	}
}

// combines all all given different-level signatures into one signature
// that has a bitset's size equal to the highest level given + 1. The +1 is
// necessary because it covers the whole space in the bitset of all signatures
// together, while the max level only covers its respective signature.
func (c *binomialPartitioner) combine(sigs []*sigPair, nbs func(int) BitSet) *sigPair {
	if len(sigs) == 0 {
		return nil
	}
	// first, find the range covering all signatures (including potentially
	// missing ones)
	// i.e. if you have level 0 and 2, then the range covering everything is
	// [min, max] where min = minimum of the range of all levels between 0 and 2
	// included, and max = max of the range of all levels between 0 and 2
	// included. Or we can just take the "inverse" range of the next level that
	// covers all levels below :)
	var maxLvl int
	for _, s := range sigs {
		if maxLvl < int(s.level) {
			maxLvl = int(s.level)
		}
	}
	globalMin, globalMax, err := c.rangeLevelInverse(maxLvl + 1)
	if err != nil {
		logf(err.Error())
		return nil
	}

	// create bitset and aggregate signatures
	finalBitSet := nbs(globalMax - globalMin)
	finalSig := sigs[0].ms.Signature

	combine := func(s *sigPair) {
		// compute the offset of this signature compared to the global bitset
		// index
		min, _, _ := c.rangeLevel(int(s.level))
		offset := min - globalMin
		bs := s.ms.BitSet
		for i := 0; i < bs.BitLength(); i++ {
			finalBitSet.Set(offset+i, bs.Get(i))
		}
	}

	combine(sigs[0])
	for _, s := range sigs[1:] {
		combine(s)
		finalSig = finalSig.Combine(s.ms.Signature)
	}

	return &sigPair{
		level: byte(maxLvl + 1),
		ms: &MultiSignature{
			Signature: finalSig,
			BitSet:    finalBitSet,
		},
	}
}

// randomBinPartitioner is a Partitioner similar to binTreePartition with
// randomization.  Basically the only impacted method is `PickNextAt`: it now
// returns nodes in a candidate set in a random order.
type randomBinPartitioner struct {
	*binomialPartitioner
	r       *mathRand.Rand
	genesis [8]byte
	seeds   map[int]int64
}

// NewRandomBinPartitioner returns a randomBinPartitioner initialized with the
// given seed. If the seed is nil, it reads from Golang's cryptographically secure
// random source with `crypto.Read`.
func NewRandomBinPartitioner(id int32, reg Registry, seed []byte) Partitioner {
	b := NewBinPartitioner(id, reg)
	if seed == nil {
		seed = make([]byte, 8)
		cryptoRand.Read(seed)
	}
	var source [8]byte
	copy(source[:], seed)
	rnd := mathRand.New(&cryptoSource{})
	return &randomBinPartitioner{
		binomialPartitioner: b.(*binomialPartitioner),
		r:                   rnd,
		genesis:             source,
		seeds:               computeSeeds(b.MaxLevel(), rnd),
	}
}

// PickNextAt implements the partitioner interface but returns randomized slice
// of identities. It keeps track of the last seen id in the randomized list.
func (r *randomBinPartitioner) PickNextAt(level, count int) ([]Identity, bool) {
	if level <= 0 {
		panic("Wrong level number")
	}

	min, max, err := r.rangeLevel(level)
	if err != nil {
		panic(err)
	}

	cardinality := max - min

	// the picked map is used differently than in binTreePartitioner -
	// the int stored indicates the minimum index in the permutation of that
	// level we should start picking identities again.
	minPicked, ok := r.picked[level]
	if !ok {
		minPicked = 0
	}
	if minPicked == max {
		return nil, false
	}


	seed, ok := r.seeds[level]
	if !ok {
		panic("random bin. tree: seed not found - internal error")
	}

	upTo := minPicked + count
	if upTo > cardinality {
		upTo = cardinality
	}

	rnd := mathRand.New(mathRand.NewSource(seed))
	perm := rnd.Perm(cardinality)
	ids := make([]Identity, 0, count)
	for i := minPicked; i < upTo; i++ {
		// take the randomized index of the sublist
		randomPermIndex := perm[i]
		// compute the global index of the Identity we want
		globalIndex := min + randomPermIndex
		randomID, ok := r.reg.Identity(globalIndex)
		if !ok {
			continue
		}
		ids = append(ids, randomID)
	}
	r.picked[level] = upTo
	return ids, true
}

func computeSeeds(levels int, r *rand.Rand) map[int]int64 {
	m := make(map[int]int64)
	for i := 0; i <= levels; i++ {
		m[i] = r.Int63()
	}
	return m
}

// random permutation based on /dev/urandom
// taken from
// https://stackoverflow.com/questions/40965044/using-crypto-rand
// -for-generating-permutations-with-rand-perm
type cryptoSource [8]byte

func (s *cryptoSource) Int63() int64 {
	cryptoRand.Read(s[:])
	return int64(binary.BigEndian.Uint64(s[:]) & (1<<63 - 1))
}

func (s *cryptoSource) Seed(seed int64) {
	panic("seed")
}
