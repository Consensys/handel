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

	// UpdatePeriod indicates at which frequency a Handel nodes sends updates
	// about its state to other Handel nodes.
	UpdatePeriod time.Duration

	// UpdateCount indicates the number of nodes contacted during each update at
	// a given level.
	UpdateCount int

	// CandidateCount indicates how many peers should we contact each time we
	// send packets to Handel nodes in a given candidate set. New nodes are
	// selected each time but no more than CandidateCount.
	CandidateCount int

	// NewBitSet returns an empty bitset. This function is used to parse
	// incoming packets containing bitsets.
	NewBitSet func(bitlength int) BitSet

	// NewPartitioner returns the Partitioner to use for this Handel round. If
	// nil, it returns the RandomBinPartitioner. The id is the ID Handel is
	// responsible for and reg is the global registry of participants.
	NewPartitioner func(id int32, reg Registry) Partitioner

	// NewEvaluatorStrategy returns the signature evaluator to use during the
	// Handel round.
	NewEvaluatorStrategy func(s signatureStore, h *Handel) SigEvaluator

	// NewTimeoutStrategy returns the Timeout strategy to use during the Handel
	// round. By default, it uses the linear timeout strategy.
	NewTimeoutStrategy func(*Handel) TimeoutStrategy
}

// DefaultConfig returns a default configuration for Handel.
func DefaultConfig(size int) *Config {
	return &Config{
		ContributionsPerc:    DefaultContributionsPerc,
		CandidateCount:       DefaultCandidateCount,
		UpdatePeriod:         DefaultUpdatePeriod,
		UpdateCount:          DefaultUpdateCount,
		NewBitSet:            DefaultBitSet,
		NewPartitioner:       DefaultPartitioner,
		NewEvaluatorStrategy: DefaultEvaluatorStrategy,
		NewTimeoutStrategy:   DefaultTimeoutStrategy,
	}
}

// DefaultContributionsPerc is the default percentage used as the required
// number of contributions in a multi-signature.
const DefaultContributionsPerc = 51

// DefaultCandidateCount is the default candidate count used by Handel.
const DefaultCandidateCount = 10

// DefaultUpdatePeriod is the default update period used by Handel.
const DefaultUpdatePeriod = 20 * time.Millisecond

// DefaultUpdateCount is the default number of candidate contacted during an
// update
const DefaultUpdateCount = 1

// DefaultBitSet returns the default implementation used by Handel, i.e. the
// WilffBitSet
var DefaultBitSet = func(bitlength int) BitSet { return NewWilffBitset(bitlength) }

// DefaultPartitioner returns the default implementation of the Partitioner used
// by Handel, i.e. RandomBinPartitioner.
var DefaultPartitioner = func(id int32, reg Registry) Partitioner {
	return NewRandomBinPartitioner(id, reg, nil)
}

// DefaultEvaluatorStrategy returns an evaluator based on the store's own
// evaluation strategy.
var DefaultEvaluatorStrategy = func(store signatureStore, h *Handel) SigEvaluator {
	return newEvaluatorStore(store)
}

// DefaultTimeoutStrategy returns the default timeout strategy used by handel -
// the linear strategy with the default timeout. See DefaultLevelTimeout.
func DefaultTimeoutStrategy(h *Handel) TimeoutStrategy {
	return NewDefaultLinearTimeout(h)
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
	if c.UpdatePeriod == 0*time.Second {
		c2.UpdatePeriod = DefaultUpdatePeriod
	}
	if c.UpdateCount == 0 {
		c2.UpdateCount = DefaultUpdateCount
	}
	if c.NewBitSet == nil {
		c2.NewBitSet = DefaultBitSet
	}
	if c.NewPartitioner == nil {
		c2.NewPartitioner = DefaultPartitioner
	}
	if c.NewEvaluatorStrategy == nil {
		c2.NewEvaluatorStrategy = DefaultEvaluatorStrategy
	}
	if c.NewTimeoutStrategy == nil {
		c2.NewTimeoutStrategy = DefaultTimeoutStrategy
	}
	return &c2
}
