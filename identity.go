package handel

// Identity holds the public informations of a Handel node
type Identity interface {
	Address() string
	// PublicKey returns the public key associated with that given node
	PublicKey() PublicKey
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
