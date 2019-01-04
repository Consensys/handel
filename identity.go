package handel

import "fmt"

// IDSIZE of the ID used in Handel. This is fixed at the moment.
const IDSIZE = 32

// Identity holds the public informations of a Handel node
type Identity interface {
	// Address must be understandable by the Network implementation
	Address() string
	// PublicKey returns the public key associated with that given node
	PublicKey() PublicKey
	// ID returns the ID used by handel to denote and classify nodes. It is best
	// if the IDs are continuous over a given finite range.
	ID() int32
}

// Registry abstracts the bookeeping of the list of Handel nodes
type Registry interface {
	// Size returns the total number of Handel nodes
	Size() int
	// Identity returns the identity at this index in the registry, or
	// (nil,false) if the index is out of bound.
	Identity(int) (Identity, bool)
	// Identities is similar to Identity but returns an array instead that
	// includes nodes whose IDs are between from inclusive and to exclusive.
	Identities(from, to int) ([]Identity, bool)
}

// fixedIdentity is an Identity that takes fixed argument
type fixedIdentity struct {
	id   int32
	addr string
	p    PublicKey
}

// NewStaticIdentity returns an Identity fixed by these parameters
func NewStaticIdentity(id int32, addr string, p PublicKey) Identity {
	return &fixedIdentity{
		id:   id,
		addr: addr,
		p:    p,
	}
}

func (s *fixedIdentity) Address() string {
	return s.addr
}

func (s *fixedIdentity) ID() int32 {
	return s.id
}

func (s *fixedIdentity) PublicKey() PublicKey {
	return s.p
}

func (s *fixedIdentity) String() string {
	if s.addr == "" {
		return fmt.Sprintf("{id:%d}", s.id)
	}
	return fmt.Sprintf("{id: %d - %s}", s.id, s.addr)
}

// arrayRegistry is a Registry that uses a fixed size array as backend
type arrayRegistry struct {
	ids []Identity
}

// NewArrayRegistry returns a Registry that uses a fixed size array as backend
func NewArrayRegistry(ids []Identity) Registry {
	return &arrayRegistry{
		ids: ids,
	}
}

func (a *arrayRegistry) Size() int {
	return len(a.ids)
}

func (a *arrayRegistry) Identity(idx int) (Identity, bool) {
	if idx < 0 || idx >= len(a.ids) {
		return nil, false
	}
	return a.ids[idx], true
}

func (a *arrayRegistry) Identities(from, to int) ([]Identity, bool) {
	if !a.inBound(from) || !a.inBound(to) {
		return nil, false
	}
	if to < from {
		return nil, false
	}
	return a.ids[from:to], true
}

func (a *arrayRegistry) inBound(idx int) bool {
	return !(idx < 0 || idx > len(a.ids))
}
