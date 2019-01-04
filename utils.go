package handel

import (
	"fmt"
	"math"
)

func log2(size int) int {
	r := math.Log2(float64(size))
	return int(math.Ceil(r))
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func pow2(n int) int {
	return int(math.Pow(2, float64(n)))
}

func isContained(arr []int, v int) bool {
	for _, v2 := range arr {
		if v2 == v {
			return true
		}
	}
	return false
}

// isSet returns true if the bit is set to 1 at the given index in the binary
// form of nb
func isSet(nb, index uint) bool {
	return ((nb >> index) & 1) == 1
}

func addresses(ids []Identity) []string {
	a := make([]string, len(ids))
	for i, id := range ids {
		a[i] = id.Address()
	}
	return a
}

// PrintLog makes logf print all statements if it is true. If false, no log are
// outputted.
var PrintLog = true

func logf(s string, args ...interface{}) {
	if PrintLog {
		fmt.Printf(s+"\n", args...)
	}
}

func combinePubKeys(c Constructor, keys []Identity) PublicKey {
	pub := c.PublicKey()
	for _, id := range keys {
		pub = pub.Combine(id.PublicKey())
	}
	return pub
}
