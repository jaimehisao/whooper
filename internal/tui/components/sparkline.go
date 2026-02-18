package components

import "math"

// Sparkline renders a unicode block sparkline from the given data.
// The width parameter controls how many characters wide the output is.
// If len(data) > width, data is downsampled; if less, it is used as-is.
func Sparkline(data []float64, width int) string {
	if len(data) == 0 || width <= 0 {
		return ""
	}

	blocks := []rune("▁▂▃▄▅▆▇█")
	levels := float64(len(blocks) - 1)

	// Downsample or use data directly.
	values := data
	if len(data) > width {
		values = downsample(data, width)
	}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	span := max - min
	if span == 0 {
		span = 1
	}

	out := make([]rune, len(values))
	for i, v := range values {
		normalized := (v - min) / span
		idx := int(math.Round(normalized * levels))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		out[i] = blocks[idx]
	}
	return string(out)
}

func downsample(data []float64, width int) []float64 {
	result := make([]float64, width)
	bucketSize := float64(len(data)) / float64(width)
	for i := 0; i < width; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(data) {
			end = len(data)
		}
		if start >= end {
			if start < len(data) {
				result[i] = data[start]
			}
			continue
		}
		sum := 0.0
		for j := start; j < end; j++ {
			sum += data[j]
		}
		result[i] = sum / float64(end-start)
	}
	return result
}
