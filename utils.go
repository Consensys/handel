package handel

import "math"

func log2(size int) int {
	r := math.Log2(float64(size))
	return int(math.Ceil(r))
}
