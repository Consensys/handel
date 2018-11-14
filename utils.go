package handel

import (
	"fmt"
	"math"
)

func log2(size int) int {
	r := math.Log2(float64(size))
	return int(math.Ceil(r))
}

// isSet returns true if the bit is set to 1 at the given index in the binary
// form of nb
func isSet(nb, index uint) bool {
	return ((nb >> index) & 1) == 1
}

// PrintLog makes logf print all statements if it is true. If false, no log are
// outputted.
var PrintLog = true

func logf(s string, args ...interface{}) {
	if PrintLog {
		fmt.Printf(s, args...)
	}
}
