package analysis

// MovingAverage computes a simple moving average over a window size.
func MovingAverage(data []float64, window int) []float64 {
	if window <= 0 || len(data) == 0 {
		return nil
	}
	result := make([]float64, len(data))
	sum := 0.0
	for i, v := range data {
		sum += v
		if i >= window {
			sum -= data[i-window]
		}
		w := window
		if i+1 < w {
			w = i + 1
		}
		result[i] = sum / float64(w)
	}
	return result
}

// PercentChange returns the percent change between consecutive values.
func PercentChange(data []float64) []float64 {
	if len(data) < 2 {
		return nil
	}
	result := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		if data[i-1] == 0 {
			result[i-1] = 0
		} else {
			result[i-1] = (data[i] - data[i-1]) / data[i-1] * 100
		}
	}
	return result
}
