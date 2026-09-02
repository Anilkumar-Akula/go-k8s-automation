// Package promscrape fetches and parses one controller's /metrics
// endpoint (Prometheus text exposition format) and offers small
// accessors over the result — enough to read the specific counters and
// gauges each controller exposes, without pulling in a full Prometheus
// server or query language.
package promscrape

import (
	"context"
	"fmt"
	"net/http"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Families is a parsed /metrics response, keyed by metric name.
type Families map[string]*dto.MetricFamily

var httpClient = &http.Client{Timeout: 5 * time.Second}

// Fetch retrieves and parses the Prometheus text exposition format at url.
func Fetch(ctx context.Context, url string) (Families, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scrape %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scrape %s: status %d", url, resp.StatusCode)
	}

	// expfmt.TextParser{} zero value has an unset name-validation scheme
	// and panics on first use; the controllers emit standard ASCII
	// metric names, so legacy validation is what we want.
	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", url, err)
	}
	return Families(families), nil
}

// CounterSum returns the sum of every series of a counter (or the raw
// value of an unlabeled one), 0 if the metric wasn't present.
func (f Families) CounterSum(name string) float64 {
	mf, ok := f[name]
	if !ok {
		return 0
	}
	var sum float64
	for _, m := range mf.GetMetric() {
		sum += m.GetCounter().GetValue()
	}
	return sum
}

// GaugeSum is CounterSum for gauges.
func (f Families) GaugeSum(name string) float64 {
	mf, ok := f[name]
	if !ok {
		return 0
	}
	var sum float64
	for _, m := range mf.GetMetric() {
		sum += m.GetGauge().GetValue()
	}
	return sum
}

// Series is one labeled time series' label set and value.
type Series struct {
	Labels map[string]string
	Value  float64
}

// CounterSeries returns every labeled series of a counter vec.
func (f Families) CounterSeries(name string) []Series {
	return series(f, name, func(m *dto.Metric) float64 { return m.GetCounter().GetValue() })
}

// GaugeSeries returns every labeled series of a gauge vec.
func (f Families) GaugeSeries(name string) []Series {
	return series(f, name, func(m *dto.Metric) float64 { return m.GetGauge().GetValue() })
}

func series(f Families, name string, value func(*dto.Metric) float64) []Series {
	mf, ok := f[name]
	if !ok {
		return nil
	}
	out := make([]Series, 0, len(mf.GetMetric()))
	for _, m := range mf.GetMetric() {
		labels := make(map[string]string, len(m.GetLabel()))
		for _, l := range m.GetLabel() {
			labels[l.GetName()] = l.GetValue()
		}
		out = append(out, Series{Labels: labels, Value: value(m)})
	}
	return out
}
