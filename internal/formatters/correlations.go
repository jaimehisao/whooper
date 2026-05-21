package formatters

import (
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type CorrelationDisplayData struct {
	XMetric  string
	YMetric  string
	Points   []CorrelationPoint
	PearsonR float64
	HasData  bool
}

type CorrelationPoint struct {
	X float64
	Y float64
}

var correlationMetrics = map[string]bool{
	"recovery":         true,
	"hrv":              true,
	"rhr":              true,
	"strain":           true,
	"sleep_duration":   true,
	"sleep_efficiency": true,
}

func FormatCorrelationData(points []store.CorrelationPoint, xMetric, yMetric string) CorrelationDisplayData {
	result := CorrelationDisplayData{
		XMetric: xMetric,
		YMetric: yMetric,
		HasData: len(points) > 0,
	}

	if len(points) == 0 {
		return result
	}

	for _, p := range points {
		result.Points = append(result.Points, CorrelationPoint{X: p.X, Y: p.Y})
	}

	return result
}

func ValidateMetric(metric string) bool {
	return correlationMetrics[metric]
}

func AvailableMetrics() []string {
	return []string{
		"recovery",
		"hrv",
		"rhr",
		"strain",
		"sleep_duration",
		"sleep_efficiency",
	}
}

func CorrelationDescription(xMetric, yMetric string, r float64) string {
	strength := "weak"
	if r > 0.7 || r < -0.7 {
		strength = "strong"
	} else if r > 0.4 || r < -0.4 {
		strength = "moderate"
	}

	direction := "positive"
	if r < 0 {
		direction = "negative"
	}

	return strength + " " + direction + " correlation between " + xMetric + " and " + yMetric + " (r=" + formatFloatSimple(r) + ")"
}

func formatFloatSimple(f float64) string {
	if f < 0 {
		return "-"
	}
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 100)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	return string(rune('0'+byte(intPart))) + "." + string(rune('0'+byte(fracPart/10))) + string(rune('0'+byte(fracPart%10)))
}
