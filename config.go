package handel

import (
	"math"
	"time"
)

// Config holds the different parameters used to configure Handel.
type Config struct {
	// ContributionsPerc is the percentage of contributions a multi-signature
	// must contain to be considered as valid. Handel will only output
	// multi-signature containing more than this threshold of contributions.
	// It must be typically above 50% of the number of Handel nodes. If not
	// specified, 50% is used by default. This percentage is used to decide when
	// a multi-signature can be passed up to higher levels as well, not only for
	// the final level.
	ContributionsPerc int

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

	// NewBitSet returns an empty bitset. This function is used to parse
	// incoming packets containing bitsets.
	NewBitSet func(bitlength int) BitSet

	// NewPartitioner returns the Partitioner to use for this Handel round. If
	// nil, it returns the RandomBinPartitioner. The id is the ID Handel is
	// responsible for and reg is the global registry of participants.
	NewPartitioner func(id int32, reg Registry) Partitioner
}

// DefaultConfig returns a default configuration for Handel.
func DefaultConfig(size int) *Config {
	return &Config{
		ContributionsPerc: DefaultContributionsPerc,
		CandidateCount:    DefaultCandidateCount,
		LevelTimeout:      DefaultLevelTimeout,
		UpdatePeriod:      DefaultUpdatePeriod,
		NewBitSet:         DefaultBitSet,
		NewPartitioner:    DefaultPartitioner,
	}
}

// DefaultContributionsPerc is the default percentage used as the required
// number of contributions in a multi-signature.
const DefaultContributionsPerc = 51

// DefaultLevelTimeout is the default level timeout used by Handel.
const DefaultLevelTimeout = 300 * time.Millisecond

// DefaultCandidateCount is the default candidate count used by Handel.
const DefaultCandidateCount = 10

// DefaultUpdatePeriod is the default update period used by Handel.
const DefaultUpdatePeriod = 200 * time.Millisecond

// DefaultBitSet returns the default implementation used by Handel, i.e. the
// WilffBitSet
var DefaultBitSet = func(bitlength int) BitSet { return NewWilffBitset(bitlength) }

// DefaultPartitioner returns the default implementation of the Partitioner used
// by Handel, i.e. RandomBinPartitioner.
var DefaultPartitioner = func(id int32, reg Registry) Partitioner {
	return NewRandomBinPartitioner(id, reg, nil)
}

// ContributionsThreshold returns the threshold of contributions required in a
// multi-signature to be considered valid and be passed up to the application
// using Handel. Basically multiplying the total number of node times the
// contributions percentage.
func (c *Config) ContributionsThreshold(n int) int {
	return int(math.Ceil(float64(n) * float64(c.ContributionsPerc) / 100.0))
}

func mergeWithDefault(c *Config, size int) *Config {
	c2 := *c
	if c.ContributionsPerc == 0 {
		c2.ContributionsPerc = DefaultContributionsPerc
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
	if c.NewBitSet == nil {
		c2.NewBitSet = DefaultBitSet
	}
	if c.NewPartitioner == nil {
		c2.NewPartitioner = DefaultPartitioner
	}
	return &c2
}
