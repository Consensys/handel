package handel

import "math"

func log2(size int) int {
	r := math.Log2(float64(size))
	return int(math.Ceil(r))
}

// isSet returns true if the bit is set to 1 at the given index in the binary
// form of nb
func isSet(nb, index uint) bool {
	return ((nb >> index) & 1) == 1
}
