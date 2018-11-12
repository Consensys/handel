package handel

import "time"

// Config holds the different parameters used to configure Handel.
type Config struct {
	// ContributionsThreshold is the threshold of contributions the multi-signature
	// must contain to be considered as valid. Handel will only output
	// multi-signature containing more than this threshold of contributions.
	// It must be typically above 50% of the number of Handel nodes. If not
	// specified, 50% is used by default.
	ContributionsThreshold int

	// LevelTimeout is used to decide when a Handel nodes passes to the next
	// level even if it did not receive enough signatures. If not specified, a
	// timeout of 500ms is used by default.
	LevelTimeout time.Duration

	// CandidateCount indicates how many peers should we contact each time we
	// send packets to Handel nodes in a given candidate set. New nodes are
	// selected each time but no more than CandidateCount.
	CandidateCount int

	// UpdatePeriod indicates at which frequency a Handel nodes sends updates
	// about its state to other Handel nodes.
	UpdatePeriod time.Duration
}

// DefaultConfig returns a default configuration for Handel.
func DefaultConfig(size int) *Config {
	return &Config{
		ContributionsThreshold: DefaultContributionsThreshold(size),
		CandidateCount:         DefaultCandidateCount,
		LevelTimeout:           DefaultLevelTimeout,
		UpdatePeriod:           DefaultUpdatePeriod,
	}
}

// DefaultContributionsThreshold returns the default contributions threshold.
func DefaultContributionsThreshold(size int) int {
	panic("not implemented yet")
}

// DefaultLevelTimeout is the default level timeout used by Handel.
const DefaultLevelTimeout = 300 * time.Millisecond

// DefaultCandidateCount is the default candidate count used by Handel.
const DefaultCandidateCount = 10

// DefaultUpdatePeriod is the default update period used by Handel.
const DefaultUpdatePeriod = 50 * time.Millisecond

func mergeWithDefault(c *Config, size int) *Config {
	c2 := *c
	if c.ContributionsThreshold == 0 {
		c2.ContributionsThreshold = DefaultContributionsThreshold(size)
	}
	if c.CandidateCount == 0 {
		c2.CandidateCount = DefaultCandidateCount
	}
	if c.LevelTimeout == 0*time.Second {
		c2.LevelTimeout = DefaultLevelTimeout
	}
	if c.UpdatePeriod == 0*time.Second {
		c2.UpdatePeriod = DefaultUpdatePeriod
	}
	return &c2
}
