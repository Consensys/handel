package handel

import "fmt"

// BlackListStrategy defines strategy for handling dishonest/broken peers.
// Implementation must be thread-safe.
// Examples of BlackListStrategy:
// if the peer sends N invalid signatures during some time interval T, then add peer to the balcklist.
type BlackListStrategy interface {
	// Update updates blacklist
	Update(id int32, err error)
	// IsBlackListed returns true if peer is blacklisted
	IsBlackListed(id int32) bool
}

// Default BlackListStrategy assumes that Peer ID is the same as packet.Origin,
// this assumption is not safe as malicious peers can pretend to be somone else, but
// is good enough for gathering basic statistics/reporting.
// For network transport which supports authetication Peer ID can be obtained
// from session details.
type defaultBlackList struct{}

func newDefaultBlackListStrategy() BlackListStrategy {
	return &defaultBlackList{}
}

func (*defaultBlackList) Update(id int32, err error) {
	fmt.Println("Signature received from peer", id, "is invalid:", err)
}

func (*defaultBlackList) IsBlackListed(id int32) bool {
	return false
}
