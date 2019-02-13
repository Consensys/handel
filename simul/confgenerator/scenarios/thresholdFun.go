package scenarios

import "math"

// CalcThreshold100 returns the same number given
func CalcThreshold100(nodes int) int {
	return nodes
}

// CalcThreshold80 returns 80% of the number given
func CalcThreshold80(nodes int) int {
	return CalcThreshold(nodes, 0.8)
}

// CalcThreshold51 returns 51% of the number given
func CalcThreshold51(nodes int) int {
	return CalcThreshold(nodes, 0.51)
}

// CalcThreshold returns the number * t
func CalcThreshold(nodes int, t float64) int {
	return int(math.Ceil(float64(nodes) * t))
}
