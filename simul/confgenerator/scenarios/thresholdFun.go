package scenarios

import "math"

func CalcThreshold100(nodes int) int {
	return nodes
}

func CalcThreshold80(nodes int) int {
	return calcThreshold(nodes, 0.8)
}

func CalcThreshold51(nodes int) int {
	return calcThreshold(nodes, 0.51)
}

func calcThreshold(nodes int, t float64) int {
	return int(math.Ceil(float64(nodes) * t))
}
