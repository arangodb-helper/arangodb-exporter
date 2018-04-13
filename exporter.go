//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"

	driver "github.com/arangodb/go-driver"
	driver_http "github.com/arangodb/go-driver/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	namespace = "arangodb" // For Prometheus metrics.

)

// metricKey returns a key into the map of metrics for the given figure & group.
func metricKey(group StatisticGroup, figure StatisticFigure, postfix string) string {
	result := strings.Replace(strings.ToLower(group.Name+"_"+figure.Name), " ", "_", -1)
	if postfix != "" {
		result = result + postfix
	}
	if figure.Units != "" {
		return result + "_" + strings.ToLower(figure.Units)
	}
	return result
}

// newMetric creates one or more metrics for the given figure & group.
func newMetric(group StatisticGroup, figure StatisticFigure) []prometheus.Collector {
	switch figure.Type {
	case FigureTypeDistribution:
		return []prometheus.Collector{
			// _sum
			prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      metricKey(group, figure, "_sum"),
				Help:      figure.Description,
			}),
			// _count
			prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      metricKey(group, figure, "_count"),
				Help:      figure.Description,
			}),
			// _bucket
			prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      metricKey(group, figure, "_bucket"),
				Help:      figure.Description,
			}, []string{"le"}),
		}
	default:
		return []prometheus.Collector{
			prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      metricKey(group, figure, ""),
				Help:      figure.Description,
			}),
		}
	}
}

// Exporter collects ArangoDB statistics from the given endpoint and exports them using
// the prometheus metrics package.
type Exporter struct {
	conn    driver.Connection
	timeout time.Duration
	mutex   sync.RWMutex

	metrics                     map[string][]prometheus.Collector
	up                          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter
}

// NewExporter returns an initialized Exporter.
func NewExporter(arangodbEndpoint, jwtSecret string, sslVerify bool, timeout time.Duration) (*Exporter, error) {
	connCfg := driver_http.ConnectionConfig{
		Endpoints: []string{arangodbEndpoint},
	}
	if !sslVerify {
		connCfg.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, err := driver_http.NewConnection(connCfg)
	if err != nil {
		return nil, maskAny(err)
	}
	if jwtSecret != "" {
		hdr, err := CreateArangodJwtAuthorizationHeader(jwtSecret)
		if err != nil {
			return nil, maskAny(err)
		}
		auth := driver.RawAuthentication(hdr)
		conn, err = conn.SetAuthentication(auth)
		if err != nil {
			return nil, maskAny(err)
		}
	}

	return &Exporter{
		conn:    conn,
		timeout: timeout,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Was the last scrape of ArangoDB successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_total_scrapes",
			Help:      "Current total ArangoDB scrapes.",
		}),
		failedScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exporter_failed_scrapes",
			Help:      "Number of failed ArangoDB scrapes",
		}),
		metrics: make(map[string][]prometheus.Collector),
	}, nil
}

// Describe describes all the metrics ever exported by the HAProxy exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, ms := range e.metrics {
		for _, m := range ms {
			m.Describe(ch)
		}
	}
	ch <- e.up.Desc()
	ch <- e.totalScrapes.Desc()
	ch <- e.failedScrapes.Desc()
}

// Collect fetches the stats from ArangoDB statistics and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()
	e.resetMetrics()
	e.scrape(ctx)

	ch <- e.up
	ch <- e.totalScrapes
	ch <- e.failedScrapes
	e.collectMetrics(ch)
}

// scrape performs a single query of all statistics.
func (e *Exporter) scrape(ctx context.Context) {
	e.totalScrapes.Inc()

	// Gather descriptions
	descr, err := GetStatisticsDescription(ctx, e.conn)
	if err != nil {
		e.up.Set(0)
		log.Errorf("Failed to fetch statistic descriptions: %v", err)
		return
	}

	// Collect statistics
	stats, err := GetStatistics(ctx, e.conn)
	if err != nil {
		e.up.Set(0)
		log.Errorf("Failed to fetch statistics: %v", err)
		return
	}

	// Mark ArangoDB as up.
	e.up.Set(1)

	// Now parse the statistics & put them in the correct metrics
	groups := make(map[string]StatisticGroup)
	for _, g := range descr.Groups {
		groups[g.Group] = g
	}
	for _, f := range descr.Figures {
		group, found := groups[f.Group]
		if !found {
			// Skip figure with unknown group
			continue
		}
		groupStats := stats.GetGroup(f.Group)
		if groupStats == nil {
			// Skip no group is found in the statistics
		}
		key := metricKey(group, f, "")
		ms, found := e.metrics[key]
		if !found {
			ms = newMetric(group, f)
			e.metrics[key] = ms
		}
		switch f.Type {
		case FigureTypeCurrent, FigureTypeAccumulated:
			if value, ok := groupStats.GetFloat(f.Identifier); ok {
				gauge := ms[0].(prometheus.Gauge)
				gauge.Set(value)
			}
		case FigureTypeDistribution:
			distStats := groupStats.GetGroup(f.Identifier)
			if distStats != nil {
				// _sum comes first
				if sum, ok := distStats.GetFloat("sum"); ok {
					gauge := ms[0].(prometheus.Gauge)
					gauge.Set(sum)
				}
				// _count comes second
				if sum, ok := distStats.GetFloat("count"); ok {
					gauge := ms[1].(prometheus.Gauge)
					gauge.Set(sum)
				}
				// _bucket comes third
				if counts, ok := distStats.GetCounts("counts"); ok {
					gaugeVec := ms[2].(*prometheus.GaugeVec)
					cummulative := int64(0)
					for i, v := range counts {
						var leValue string
						if i < len(f.Cuts) {
							leValue = fmt.Sprintf("%v", f.Cuts[i])
						} else {
							leValue = "+Inf"
						}
						gaugeVec.WithLabelValues(leValue).Set(float64(cummulative + v))
						cummulative += v
					}
				}
			}
		}
	}
}

type resetter interface {
	Reset()
}

func (e *Exporter) resetMetrics() {
	for _, ms := range e.metrics {
		for _, m := range ms {
			if r, ok := m.(resetter); ok {
				r.Reset()
			}
		}
	}
}

func (e *Exporter) collectMetrics(metrics chan<- prometheus.Metric) {
	for _, ms := range e.metrics {
		for _, m := range ms {
			if c, ok := m.(prometheus.Collector); ok {
				c.Collect(metrics)
			}
		}
	}
}
