package handel

import (
	"bytes"
	"encoding/binary"

	"github.com/willf/bitset"
)

// BitSet interface. Available implementation is a wrapper around wilff's bitset
// library.
type BitSet interface {
	// BitLength returns the fixed size of this BitSet
	BitLength() int
	// Cardinality returns the number of '1''s set
	Cardinality() int
	// Set the bit at the given index to 1 or 0 depending on the given boolean.
	// A set out of bounds is an error, implementations should panic in such a case.
	Set(int, bool)
	// Get returns the status of the i-th bit in this bitset.
	// A get out of bounds is an error, implementations should panic in such a case.
	Get(int) bool
	// MarshalBinary returns the binary representation of the BitSet.
	MarshalBinary() ([]byte, error)
	// UnmarshalBinary fills the bitset from the given buffer.
	UnmarshalBinary([]byte) error
	// returns the binary representation of this bitset in string
	String() string
	// All returns true if all bits are set, false otherwise. Returns true for
	// empty sets.
	All() bool
	// None returns true if no bit is set, false otherwise. Returns true for
	// empty sets.
	None() bool
	// Any returns true if any bit is set, false otherwise
	Any() bool
	// Or between this bitset and another, returns a new bitset.
	Or(b2 BitSet) BitSet
	// And between this bitset and another, returns a new bitset.
	And(b2 BitSet) BitSet
	// Xor between this bitset and another, returns a new bitset.
	Xor(b2 BitSet) BitSet
	// IsSuperSet returns true if this is a superset of the other set
	IsSuperSet(b2 BitSet) bool
	// NextSet returns the next bit set from the specified index,
	// including possibly the current index
	// along with an error code (true = valid, false = no set bit found)
	// for i,e := v.NextSet(0); e; i,e = v.NextSet(i + 1) {...}
	NextSet(i int) (int, bool)
	// IntersectionCardinality computes the cardinality of the differnce
	IntersectionCardinality(b2 BitSet) int
	// Clone this BitSet
	Clone() BitSet
}

// WilffBitSet implements a BitSet using the wilff library.
type WilffBitSet struct {
	b *bitset.BitSet
	l int
}

// NewWilffBitset returns a BitSet implemented using the wilff's bitset library.
func NewWilffBitset(length int) BitSet {
	return &WilffBitSet{
		b: bitset.New(uint(length)),
		l: length,
	}
}

func newWilffBitset(bs *bitset.BitSet) BitSet {
	return &WilffBitSet{
		b: bs,
		l: int(bs.Len()),
	}
}

// BitLength implements the BitSet interface
func (w *WilffBitSet) BitLength() int {
	return int(w.l)
}

// Cardinality implements the BitSet interface
func (w *WilffBitSet) Cardinality() int {
	return int(w.b.Count())
}

// Set implements the BitSet interface
func (w *WilffBitSet) Set(idx int, status bool) {
	if !w.inBound(idx) {
		panic("bitset: set out of bounds")
	}
	w.b = w.b.SetTo(uint(idx), status)
}

// Get implements the BitSet interface
func (w *WilffBitSet) Get(idx int) bool {
	if !w.inBound(idx) {
		panic("bitset: get out of bounds")
	}
	return w.b.Test(uint(idx))
}

// Combine implements the BitSet interface
func (w *WilffBitSet) Combine(b2 BitSet) BitSet {
	// XXX Panics if used wrongly at the moment. Could be possible to use other
	// implementations by using the interface's method and implementing or
	// ourselves.
	w2 := b2.(*WilffBitSet)
	totalLength := w.l + w2.l
	w3 := NewWilffBitset(totalLength).(*WilffBitSet)

	w3.b.InPlaceUnion(w.b)
	for i := 0; i < w2.l; i++ {
		w3.Set(i+w.l, w2.Get(i))
	}
	return w
}

// Or implements the BitSet interface
func (w *WilffBitSet) Or(b2 BitSet) BitSet {
	return newWilffBitset(w.b.Union(b2.(*WilffBitSet).b))
}

// And implements the BitSet interface
func (w *WilffBitSet) And(b2 BitSet) BitSet {
	return newWilffBitset(w.b.Intersection(b2.(*WilffBitSet).b))
}

// Xor implements the BitSet interface
func (w *WilffBitSet) Xor(b2 BitSet) BitSet {
	return newWilffBitset(w.b.SymmetricDifference(b2.(*WilffBitSet).b))
}

// Clone implements the BitSet interface
func (w *WilffBitSet) Clone() BitSet {
	return newWilffBitset(w.b.Clone())
}

func (w *WilffBitSet) inBound(idx int) bool {
	return !(idx < 0 || idx >= w.l)
}

// IsSuperSet implements the BitSet interface
func (w *WilffBitSet) IsSuperSet(b2 BitSet) bool {
	return w.b.IsSuperSet(b2.(*WilffBitSet).b)
}

// MarshalBinary implements the go Marshaler interface. It encodes the size
// first and then the bitset.
func (w *WilffBitSet) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	err := binary.Write(&b, binary.BigEndian, uint16(w.l))
	if err != nil {
		return nil, err
	}
	buff, err := w.b.MarshalBinary()
	if err != nil {
		return nil, err
	}
	b.Write(buff)
	return b.Bytes(), nil
}

// UnmarshalBinary implements the go Marshaler interface. It decodes the length
// first and then the bitset.
func (w *WilffBitSet) UnmarshalBinary(buff []byte) error {
	var b = bytes.NewBuffer(buff)
	var length uint16
	err := binary.Read(b, binary.BigEndian, &length)
	if err != nil {
		return err
	}

	w.b = new(bitset.BitSet)
	w.l = int(length)
	return w.b.UnmarshalBinary(b.Bytes())
}

func (w *WilffBitSet) String() string {
	return w.b.String()
}

// All implements the BitSet interface
func (w *WilffBitSet) All() bool {
	return w.b.All()
}

// None implements the BitSet interface
func (w *WilffBitSet) None() bool {
	return w.b.None()
}

// Any implements the BitSet interface
func (w *WilffBitSet) Any() bool {
	return w.b.Any()
}

// NextSet implements the BitSet interface
func (w *WilffBitSet) NextSet(i int) (int, bool) {
	ni, res := w.b.NextSet(uint(i))
	return int(ni), res
}

// IntersectionCardinality implements the BitSet interface
func (w *WilffBitSet) IntersectionCardinality(b2 BitSet) int {
	return int(w.b.IntersectionCardinality(b2.(*WilffBitSet).b))
}
