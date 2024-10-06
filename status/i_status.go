package status

type IStatus interface {
	// TODO: verify if these methods are necessary
	// GetMin() int
	// GetCurrent() int
	// GetMax() int
	Increase(value int) int
	Decrease(value int) int
	StatusUpgrade(level int)
}

func Increase(current, max, value int) int {
	temp := current + value
	if temp > max {
		return max
	}
	return temp
}

func Decrease(current, min, value int) int {
	temp := current - value
	if temp < min {
		return min
	}
	return temp
}
