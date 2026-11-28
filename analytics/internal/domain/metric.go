package domain

import "time"

type MetricChartPoint struct {
	Value     float64
	Timestamp time.Time
}

type Metric struct {
	MetricID    string
	AvgValue    float64
	MinValue    float64
	MaxValue    float64
	ChartPoints []MetricChartPoint
}
