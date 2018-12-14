package quic

import (
	"sync"
	"testing"

	h "github.com/ConsenSys/handel"
	"github.com/stretchr/testify/require"
)

type mockSucessDialer struct{}

func (q mockSucessDialer) startDial(identity h.Identity, out chan *result) {
	out <- &result{identity.ID(), nil, false, nil}
}

type mockBlockingDialer struct {
	wg *sync.WaitGroup
}

func (q mockBlockingDialer) startDial(identity h.Identity, out chan *result) {
	if q.wg != nil {
		q.wg.Done()
	}
}

const chanSize = 10

func TestSessionManager_Success(t *testing.T) {
	sesManager := newSimpleSessionManager(mockSucessDialer{}, chanSize)
	go sesManager.start()
	idx := int32(22)
	identity := h.NewStaticIdentity(idx, "", nil)
	res := sesManager.Dial(identity)
	require.Equal(t, res.id, idx)
	require.Equal(t, res.isWaiting, false)
}

func TestSessionManager_Block(t *testing.T) {
	var wg sync.WaitGroup
	sesManager := newSimpleSessionManager(mockBlockingDialer{&wg}, chanSize)
	go sesManager.start()

	idx := int32(22)
	identity := h.NewStaticIdentity(idx, "", nil)
	wg.Add(1)
	go func() {
		sesManager.Dial(identity)
	}()
	wg.Wait()
	res := sesManager.Dial(identity)
	require.Equal(t, res.id, idx)
	require.Equal(t, res.isWaiting, true)

	res2 := sesManager.Dial(identity)
	require.Equal(t, res2.id, idx)
	require.Equal(t, res2.isWaiting, true)
}

func TestSessionManager_Update(t *testing.T) {
	identity1 := h.NewStaticIdentity(int32(22), "", nil)
	identity2 := h.NewStaticIdentity(int32(23), "", nil)
	identity3 := h.NewStaticIdentity(int32(24), "", nil)

	sesManager := newSimpleSessionManager(mockBlockingDialer{}, chanSize)
	chanMap := make(map[int32]chan *result)
	go func() {
		sesManager.Dial(identity1)
	}()
	go func() {
		sesManager.Dial(identity2)
	}()
	go func() {
		sesManager.Dial(identity3)
	}()

	sesManager.update(chanMap)
	sesManager.update(chanMap)
	sesManager.update(chanMap)

	_, ok1 := chanMap[identity1.ID()]
	require.Equal(t, ok1, true)
	_, ok2 := chanMap[identity2.ID()]
	require.Equal(t, ok2, true)
	_, ok3 := chanMap[identity3.ID()]
	require.Equal(t, ok3, true)

	sesManager.out <- &result{identity2.ID(), nil, false, nil}
	sesManager.update(chanMap)
	_, ok2 = chanMap[identity2.ID()]
	require.False(t, ok2)
}
