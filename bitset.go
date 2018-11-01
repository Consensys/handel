package handel

// BitSet is .. a bitset !
// TODO: replace it by a struct that implements that interface.
type BitSet interface {
	// BItLength returns the fixed size of this BitSet
	BitLength() int
	// Cardinality returns the number of '1''s set
	Cardinality() int
	Set(int, bool)
	Get(int) bool
	// OR with the given BitSet
	Or(BitSet) BitSet
	// Combine two bitsets together and returns a new bitset whose
	// bitlength is the sum of both's bitlengths.
	Combine(BitSet) BitSet
}
