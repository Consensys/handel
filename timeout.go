package handel

import (
	"sync"
	"time"
)

// TimeoutStrategy decides when to start a level in Handel. A basic strategy
// starts level according to a linear timeout function: level $i$ starts at time
// $i * period$. The interface is started and stopped by the Handel main logic.
type TimeoutStrategy interface {
	// Called by handel when it starts
	Start()
	// // Called by handel when it stops
	Stop()
}

// linearTimeout starts each level $i$ at time $i * period$..
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

// LinearTimeoutConstructor returns the linear timeout contructor as required
// for the Config.
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
