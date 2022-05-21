package exporter

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorcon/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

type BuildInfo struct {
	Version   string
	CommitSha string
	Date      string
}

type Exporter struct {
	sync.Mutex

	rustAddr  string
	namespace string
	conn      *websocket.Conn

	totalScrapes              prometheus.Counter
	scrapeDuration            prometheus.Summary
	targetScrapeRequestErrors prometheus.Counter

	metricDescriptions map[string]*prometheus.Desc
	options            Options

	mux       *http.ServeMux
	buildInfo BuildInfo
}

type Options struct {
	Password    string
	MetricsPath string
	Namespace   string
	Registry    *prometheus.Registry
}

func NewFacepunchRustExporter(rustAddr string, opts Options) (*Exporter, error) {
	log.Debugf("NewFacepunchRustExporter options: %#v", opts)
	e := &Exporter{
		rustAddr:  rustAddr,
		options:   opts,
		namespace: opts.Namespace,

		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: opts.Namespace,
			Name:      "exporter_scrapes_total",
			Help:      "Current total redis scrapes.",
		}),

		scrapeDuration: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: opts.Namespace,
			Name:      "exporter_scrape_duration_seconds",
			Help:      "Durations of scrapes by the exporter",
		}),

		targetScrapeRequestErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: opts.Namespace,
			Name:      "target_scrape_request_errors_total",
			Help:      "Errors in requests to the exporter",
		}),
	}

	e.metricDescriptions = map[string]*prometheus.Desc{}
	for k, desc := range map[string]struct {
		txt  string
		lbls []string
	}{
		"player_count": {txt: "The number of currently connected players."},
		"up":           {txt: "Information about the FacepunchRust client"},
	} {
		e.metricDescriptions[k] = newMetricDescr(opts.Namespace, k, desc.txt, desc.lbls)
	}

	if e.options.MetricsPath == "" {
		e.options.MetricsPath = "/metrics"
	}

	e.mux = http.NewServeMux()

	if e.options.Registry != nil {
		e.options.Registry.MustRegister(e)
		e.mux.Handle(e.options.MetricsPath, promhttp.HandlerFor(
			e.options.Registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
		))
	}

	e.mux.HandleFunc("/", e.indexHandler)
	e.mux.HandleFunc("/scrape", e.scrapeHandler)
	e.mux.HandleFunc("/health", e.healthHandler)

	return e, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range e.metricDescriptions {
		ch <- desc
	}

	ch <- e.totalScrapes.Desc()
	ch <- e.scrapeDuration.Desc()
	ch <- e.targetScrapeRequestErrors.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()
	e.totalScrapes.Inc()

	if e.rustAddr != "" {
		startTime := time.Now()
		var up float64
		if err := e.scrapeRustServer(ch); err != nil {
			e.registerConstMetricGauge(ch, "exporter_last_scrape_error", 1.0, fmt.Sprintf("%s", err))
		} else {
			up = 1
			e.registerConstMetricGauge(ch, "exporter_last_scrape_error", 0, "")
		}

		e.registerConstMetricGauge(ch, "up", up)

		took := time.Since(startTime).Seconds()
		e.scrapeDuration.Observe(took)
		e.registerConstMetricGauge(ch, "exporter_last_scrape_duration_seconds", took)
	}

	ch <- e.totalScrapes
	ch <- e.scrapeDuration
	ch <- e.targetScrapeRequestErrors
}

func (e *Exporter) scrapeRustServer(ch chan<- prometheus.Metric) (err error) {
	defer log.Debugf("scrapeRustServer() done")
	startTime := time.Now()
	e.conn, err = e.connectToRust()
	connectTookSeconds := time.Since(startTime).Seconds()
	e.registerConstMetricGauge(ch, "exporter_last_scrape_connect_time_seconds", connectTookSeconds)

	if err != nil {
		log.Errorf("Couldn't connect to rust server")
		log.Errorf("connectToRust( %s ) err: %s", e.rustAddr, err)
		return err
	}
	defer e.conn.Close()

	e.extractPlayerCount(ch)

	return nil
}

/* Player Count Metrics (TODO: we might be able to get all of this with global.stats) */

func (e *Exporter) extractPlayerCount(ch chan<- prometheus.Metric) {
	log.Debugf("extractPlayerCount()")
	playerCountOutput, err := e.doRustCmd("players")
	if err != nil {
		log.Errorf("extractPlayerCount() err: %s", err)
		return
	}
	lines := strings.Split(playerCountOutput, "\r\n")
	playerCount := len(lines) - 1
	if err != nil {
		log.Errorf("extractPlayerCount() err: %s", err)
		return
	}

	e.registerConstMetricGauge(ch, "player_count", float64(playerCount))
}

func (e *Exporter) extractSleepingPlayerCount(ch chan<- prometheus.Metric) {
	// global.sleepingusers
}

func (e *Exporter) extractQueuedPlayerCount(ch chan<- prometheus.Metric) {
	// global.queue
}
