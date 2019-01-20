package handel

import (
	"sync"
	"time"
)

// TimeoutStrategy decides when to start a level in Handel. It is started and
// stopped by the Handel structure. A basic strategy starts level according to a
// linear timeout function thanks to the Handel.StartLevel method. More advanced
// strategies could for example implement the Pre/PostProcessor interface,
// register itself as a processor to Handel, and start a level according to
// specific rules such as "all nodes answered with a 1-contribution
// multi-signature", etc.
type TimeoutStrategy interface {
	// Called by handel when it starts
	Start()
	// // Called by handel when it stops
	Stop()
}

type infiniteTimeout struct {
}

// NewInfiniteTimeout creates an InfiniteTimeout. Needs this signature
func NewInfiniteTimeout(h *Handel, lvls []int) TimeoutStrategy {
	return &infiniteTimeout{}
}

// Start implements the interface
func (l *infiniteTimeout) Start() {}

// Stop implements the interface
func (l *infiniteTimeout) Stop() {}

type linearTimeout struct {
	sync.Mutex
	newLevel func(int)
	levels   []int
	period   time.Duration
	ticker   *time.Ticker
	done     chan bool
	started  bool
}

// DefaultLevelTimeout is the default level timeout used by the linear timeout
// strategy.
const DefaultLevelTimeout = 50 * time.Millisecond

// NewDefaultLinearTimeout returns a TimeoutStrategy that starts level linearly
// with the default period of DefaultLevelTimeout.  More precisely, level i
// starts at time i * period.
func NewDefaultLinearTimeout(h *Handel, levels []int) TimeoutStrategy {
	return NewLinearTimeout(h, levels, DefaultLevelTimeout)
}

// LinearTimeoutConstructor returns the contructor to give in the Config for a
// linear timeout with the given period
func LinearTimeoutConstructor(period time.Duration) func(h *Handel, levels []int) TimeoutStrategy {
	return func(h *Handel, levels []int) TimeoutStrategy {
		return NewLinearTimeout(h, levels, period)
	}
}

// NewLinearTimeout returns a TimeoutStrategy that starts level linearly with
// the given period. More precisely, it starts level i at time i * period.
func NewLinearTimeout(h *Handel, levels []int, period time.Duration) TimeoutStrategy {
	return &linearTimeout{
		period:   period,
		newLevel: h.StartLevel,
		levels:   levels,
		done:     make(chan bool, 1),
	}
}

func (l *linearTimeout) Start() {
	l.Lock()
	defer l.Unlock()
	l.started = true
	l.ticker = time.NewTicker(l.period)
	go l.linearLevels(l.ticker.C)
}

func (l *linearTimeout) Stop() {
	l.Lock()
	defer l.Unlock()
	if !l.started {
		return
	}
	l.ticker.Stop()
	close(l.done)
}

func (l *linearTimeout) linearLevels(c <-chan time.Time) {
	idx := 0
	for idx < len(l.levels) {
		l.newLevel(l.levels[idx])
		select {
		case <-c:
			idx++
		case <-l.done:
			return
		}
	}
}
