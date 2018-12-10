package quic

import (
	"time"

	h "github.com/ConsenSys/handel"
	quic "github.com/lucas-clemente/quic-go"
)

// The handel protocol is very well suited to fire and forget type of
// transport (udp for example), with quic or other type of statefull protocol
// we need to make sure handshake period is handled appropriately.
type sessionManager interface {
	Dial(identity h.Identity) *result
}

type idAndResultChan struct {
	identity h.Identity
	retChan  chan *result
}

type result struct {
	id        int32
	session   quic.Session
	isWaiting bool
	err       error
}

type simpleSesssionManager struct {
	identities chan idAndResultChan
	out        chan *result
	dialer     Dialer
}

func newSessionManager(handshakeTimeout time.Duration) sessionManager {
	dialer := newQuicDialer(handshakeTimeout)
	chanSize := 100
	sesManager := newSimpleSessionManager(dialer, chanSize)
	go sesManager.start()
	return sesManager
}

func newSimpleSessionManager(dialer Dialer, chanSize int) *simpleSesssionManager {
	sesManager := &simpleSesssionManager{
		identities: make(chan idAndResultChan, chanSize),
		out:        make(chan *result, chanSize),
		dialer:     dialer}
	return sesManager
}

// Dial blocks until session between 2 peers is established or
// handshake timeouts. If there is another Dial call to a peer
// for which the previous Dial hasn't finished the second Dial will return
// immediately with status isWaiting set to true
func (sesManager *simpleSesssionManager) Dial(identity h.Identity) *result {
	resChan := make(chan *result)
	defer close(resChan)
	sesManager.identities <- idAndResultChan{identity, resChan}
	for x := range resChan {
		return x
	}
	//unreachable code
	return nil
}

func (sesManager *simpleSesssionManager) start() {
	chanMap := make(map[int32]chan *result)
	for {
		sesManager.update(chanMap)
	}
}

func (sesManager *simpleSesssionManager) update(chanMap map[int32]chan *result) {
	select {
	case identAndResult := <-sesManager.identities:
		identity := identAndResult.identity
		id := identity.ID()
		// chanMap contains id only if there was a call to a remote peer(id) which
		// hasn't finished yet
		if _, ok := chanMap[id]; ok {
			identAndResult.retChan <- wait(id)
			return
		}
		//Put notification channel in the chanMap
		chanMap[identity.ID()] = identAndResult.retChan
		go sesManager.dialer.startDial(identity, sesManager.out)

	case res := <-sesManager.out:
		// Dial has finished:
		//  1: notify caller
		//  2: delate notification channel from the chanMap
		id := res.id
		chanMap[id] <- res
		delete(chanMap, id)
	}
}

func wait(id int32) *result {
	return &result{id: id, session: nil, isWaiting: true, err: nil}
}
