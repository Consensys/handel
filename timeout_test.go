package handel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeoutLinear(t *testing.T) {
	n := 8
	levels := 3
	_, handels := FakeSetup(n)
	h := handels[0]

	period := 20 * time.Millisecond
	tooLong := 30 * time.Millisecond
	linear := NewLinearTimeout(h, period).(*linearTimeout)

	chNewLevel := make(chan int, 1)
	newLevel := func(level int) {
		chNewLevel <- level
	}
	linear.newLevel = newLevel

	go linear.Start()
	level := 1
	unfinished := true
	for unfinished {
		select {
		case l := <-chNewLevel:
			if l > levels {
				t.FailNow()
			}
			require.Equal(t, level, l)
			level++
		case <-time.After(tooLong):
			if level <= levels {
				require.True(t, false, "waited too long time %d", level)
			}
			unfinished = false
		}
	}

	// -1 because we increment even after the last one
	require.Equal(t, levels, level-1)
}
