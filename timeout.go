package handel

import "time"

// TimeoutStrategy decides when to start a level in Handel. It is started and
// stopped by the Handel structure. A basic strategy starts level according to a
// linear timeout function thanks to the Handel.StartLevel method. More advanced
// strategies could for example implement the Pre/PostProcessor interface,
// register itself as a processor to Handel, and start a level according to
// specific rules such as "all nodes answered with a 1-contribution
// multi-signature", etc.
type TimeoutStrategy interface {
	Start()
	Stop()
}

type linearTimeout struct {
	newLevel func(int)
	maxLevel int
	period   time.Duration
	ticker   *time.Ticker
	done     chan bool
}

// DefaultLevelTimeout is the default level timeout used by the linear timeout
// strategy.
const DefaultLevelTimeout = 100 * time.Millisecond

// NewDefaultLinearTimeout returns a TimeoutStrategy that starts level linearly
// with the default period of DefaultLevelTimeout.  More precisely, level i
// starts at time i * period.
func NewDefaultLinearTimeout(h *Handel) TimeoutStrategy {
	return NewLinearTimeout(h, DefaultLevelTimeout)
}

// NewLinearTimeout returns a TimeoutStrategy that starts level linearly with
// the given period. More precisely, it starts level i at time i * period.
func NewLinearTimeout(h *Handel, period time.Duration) TimeoutStrategy {
	return &linearTimeout{
		period:   period,
		newLevel: h.StartLevel,
		maxLevel: h.Partitioner.MaxLevel(),
		done:     make(chan bool, 1),
	}
}

func (l *linearTimeout) Start() {
	l.ticker = time.NewTicker(l.period)
	// start first level directly if not done yet
	l.newLevel(1)
	go l.linearLevels(l.ticker.C)
}

func (l *linearTimeout) Stop() {
	l.ticker.Stop()
	close(l.done)
}

func (l *linearTimeout) linearLevels(c <-chan time.Time) {
	level := 2
	for level <= l.maxLevel {
		select {
		case <-c:
			l.newLevel(level)
			level++
		case <-l.done:
			return
		}
	}
}
