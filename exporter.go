package main

import (
	"context"
	"crypto/tls"
	_ "net/http/pprof"
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

var (
	frontendLabelNames = []string{"frontend"}
	backendLabelNames  = []string{"backend"}
	serverLabelNames   = []string{"backend", "server"}
)

// metricKey returns a key into the map of metrics for the given figure & group.
func metricKey(group StatisticGroup, figure StatisticFigure) string {
	return group.Name + "_" + figure.Name
}

// newMetric creates a metric for the given figure & group.
func newMetric(group StatisticGroup, figure StatisticFigure) prometheus.Metric {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricKey(group, figure),
			Help:      figure.Description,
		},
	)
}

// Exporter collects ArangoDB statistics from the given endpoint and exports them using
// the prometheus metrics package.
type Exporter struct {
	conn    driver.Connection
	timeout time.Duration
	mutex   sync.RWMutex

	metrics                     map[string]prometheus.Metric
	up                          prometheus.Gauge
	totalScrapes, failedScrapes prometheus.Counter
}

// NewExporter returns an initialized Exporter.
func NewExporter(arangodbEndpoint string, sslVerify bool, timeout time.Duration) (*Exporter, error) {
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

	return &Exporter{
		conn: conn,
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
		metrics: make(map[string]prometheus.Metric),
	}, nil
}

// Describe describes all the metrics ever exported by the HAProxy exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.metrics {
		ch <- m.Desc()
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
		key := metricKey(group, f)
		m, found := e.metrics[key]
		if !found {
			m = newMetric(group, f)
			e.metrics[key] = m
		}
		switch f.Type {
		case FigureTypeCurrent, FigureTypeAccumulated:
			if value, ok := groupStats.GetFloat(f.Identifier); ok {
				gauge := m.(prometheus.Gauge)
				gauge.Set(value)
			}
		}
	}
}

func (e *Exporter) resetMetrics() {
	/*for _, m := range e.metrics {
		//		m.Reset()
	}*/
}

func (e *Exporter) collectMetrics(metrics chan<- prometheus.Metric) {
	for _, m := range e.metrics {
		if c, ok := m.(prometheus.Collector); ok {
			c.Collect(metrics)
		}
	}
}
