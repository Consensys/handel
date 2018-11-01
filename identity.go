package handel

// Identity holds the public informations of a Handel node
type Identity interface {
	// Index returns the index in the global list of all nodes
	Index() int
	// PublicKey returns the public key associated with that given node
	PublicKey()
}

// Registry abstracts the bookeeping of the list of Handel nodes
type Registry interface {
	// Size returns the total number of Handel nodes
	Size() int
	// Identity returns the identity at this index in the registry, or
	// (nil,false) if the index is out of bound.
	Identity(int) (Identity, bool)
	// XXX May be overkill this early in the library...
	// Identities(from, to int) ([]Identity, bool)
}
