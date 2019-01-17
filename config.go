package handel

import (
	"crypto/rand"
	"io"
	"math"
	"time"
)

// Config holds the different parameters used to configure Handel.
type Config struct {
	// Contributions is the minimum number of contributions a multi-signature
	// must contain to be considered as valid. Handel will only output
	// multi-signature containing more than this threshold of contributions.
	// It must be typically above 50% of the number of Handel nodes. If not
	// specified, DefaultContributionsPerc of the number of signers is used by default.
	Contributions int

	// UpdatePeriod indicates at which frequency a Handel nodes sends updates
	// about its state to other Handel nodes.
	UpdatePeriod time.Duration

	// UpdateCount indicates the number of nodes contacted during each update at
	// a given level.
	UpdateCount int

	// NodeCount indicates how many peers should we contact each time we
	// send packets to Handel nodes in a given candidate set. New nodes are
	// selected each time but no more than NodeCount.
	NodeCount int

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
	NewTimeoutStrategy func(h *Handel, levels []int) TimeoutStrategy

	// Logger to use for logging handel actions
	Logger Logger
	// Rand provides the source of entropy for shuffling the list of nodes that
	// Handel must contact at each level. If not set, golang's crypto/rand is
	// used.
	Rand io.Reader

	// DisableShuffling is a debugging flag to not shuffle any list of nodes - it
	// is much easier to detect pattern in bugs in this manner
	DisableShuffling bool

	// SleepTimeOnSigVerify is a test feature a sleep time (in ms) rather than actually verifying the signatures
	// Can be used to save on CPU during tests or/and to test with shorter/longer verifying time
	// Set to zero by default: no sleep time. When activated the sleep replaces the verification.
	// This sleep time is approximate and depends on golang and the os. The actual delay can be longer.
	SleepTimeOnSigVerify int
}

// DefaultConfig returns a default configuration for Handel.
func DefaultConfig(numberOfNodes int) *Config {
	contributions := PercentageToContributions(DefaultContributionsPerc, numberOfNodes)
	return &Config{
		Contributions:        contributions,
		NodeCount:            DefaultCandidateCount,
		UpdatePeriod:         DefaultUpdatePeriod,
		UpdateCount:          DefaultUpdateCount,
		NewBitSet:            DefaultBitSet,
		NewPartitioner:       DefaultPartitioner,
		NewEvaluatorStrategy: DefaultEvaluatorStrategy,
		NewTimeoutStrategy:   DefaultTimeoutStrategy,
		Logger:               DefaultLogger,
		Rand:                 rand.Reader,
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
// by Handel, i.e. BinPartitioner.
var DefaultPartitioner = func(id int32, reg Registry) Partitioner {
	return NewBinPartitioner(id, reg)
}

// DefaultEvaluatorStrategy returns an evaluator based on the store's own
// evaluation strategy.
var DefaultEvaluatorStrategy = func(store signatureStore, h *Handel) SigEvaluator {
	return newEvaluatorStore(store)
}

// DefaultTimeoutStrategy returns the default timeout strategy used by handel -
// the linear strategy with the default timeout. See DefaultLevelTimeout.
func DefaultTimeoutStrategy(h *Handel, levels []int) TimeoutStrategy {
	return NewDefaultLinearTimeout(h, levels)
}

// PercentageToContributions returns the exact number of contributions needed
// out of n contributions, from the given percentage. Useful when considering
// large scale signatures as in Handel, e.g. 51%, 75%...
func PercentageToContributions(perc, n int) int {
	return int(math.Ceil(float64(n) * float64(perc) / 100.0))
}

func mergeWithDefault(c *Config, size int) *Config {
	c2 := *c
	if c.Contributions == 0 {
		n := PercentageToContributions(DefaultContributionsPerc, size)
		c2.Contributions = n
	}
	if c.NodeCount == 0 {
		c2.NodeCount = DefaultCandidateCount
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
	if c.Logger == nil {
		c2.Logger = DefaultLogger
	}
	if c.Rand == nil {
		c2.Rand = rand.Reader
	}
	if c.DisableShuffling {
		c2.DisableShuffling = true
	}
	return &c2
}
