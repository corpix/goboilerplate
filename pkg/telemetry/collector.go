package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
)

// SelfCollector implements Collector for a single Metric so that the Metric
// collects itself. Add it as an anonymous field to a struct that implements
// Metric, and call init with the Metric itself as an argument.
type SelfCollector struct {
	self prometheus.Metric
}

// init provides the SelfCollector with a reference to the metric it is supposed
// to collect. It is usually called within the factory function to create a
// metric. See example.
func (c *SelfCollector) init(self prometheus.Metric) {
	c.self = self
}

// Describe implements Collector.
func (c *SelfCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.self.Desc()
}

// Collect implements Collector.
func (c *SelfCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- c.self
}
