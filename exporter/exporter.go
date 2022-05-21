package exporter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorcon/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
)

type ServerInfo struct {
	Hostname        string  `json:"Hostname"`
	MaxPlayers      int     `json:"MaxPlayers"`
	Players         int     `json:"Players"`
	Queued          int     `json:"Queued"`
	Joining         int     `json:"Joining"`
	EntityCount     int     `json:"EntityCount"`
	GameTime        string  `json:"GameTime"`
	Uptime          int     `json:"Uptime"`
	Map             string  `json:"Map"`
	Framerate       float64 `json:"Framerate"`
	Memory          int     `json:"Memory"`
	Collections     int     `json:"Collections"`
	NetworkIn       int     `json:"NetworkIn"`
	NetworkOut      int     `json:"NetworkOut"`
	Restarting      bool    `json:"Restarting"`
	SaveCreatedTime string  `json:"SaveCreatedTime"`
}

type ServerBuildInfo struct {
	Date int `json:"date"`
	Scm  struct {
		Type     string `json:"type"`
		ChangeId string `json:"changeid"`
		Branch   string `json:"branch"`
		Repo     string `json:"repo"`
		Comment  string `json:"comment"`
		Author   string `json:"author"`
		Date     string `json:"date"`
	} `json:"Scm"`
	Build struct {
		Id     string `json:"id"`
		Number string `json:"number"`
		Tag    string `json:"tag"`
		Url    string `json:"url"`
		Name   string `json:"name"`
		Node   string `json:"node"`
	} `json:"Build"`
	Valid bool `json:"valid"`
}

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
	BuildInfo   BuildInfo
}

func NewFacepunchRustExporter(rustAddr string, opts Options) (*Exporter, error) {
	log.Debugf("NewFacepunchRustExporter options: %#v", opts)
	e := &Exporter{
		rustAddr:  rustAddr,
		options:   opts,
		namespace: opts.Namespace,

		buildInfo: opts.BuildInfo,

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

	labels := []string{
		"build_date", "build_id", "build_number", "build_tag", "build_url", "build_name", "build_node", "build_valid",
		"build_scm_type", "build_scm_changeid", "build_scm_branch", "build_scm_repo", "build_scm_comment", "build_scm_author", "build_scm_date",
	}
	e.metricDescriptions = map[string]*prometheus.Desc{}
	for k, desc := range map[string]struct {
		txt  string
		lbls []string
	}{
		"players":             {txt: "The number of currently connected players.", lbls: labels},
		"players_queued":      {txt: "The number of players queued to connect.", lbls: labels},
		"players_joining":     {txt: "The number of players connecting.", lbls: labels},
		"server_max_players":  {txt: "Max number of players allowed to join.", lbls: labels},
		"server_entity_count": {txt: "Number of entities loaded in game.", lbls: labels},
		"server_uptime":       {txt: "How long the server has been up for.", lbls: labels},
		"server_framerate":    {txt: "Server framerate.", lbls: labels},
		"server_memory":       {txt: "Server memory consumption.", lbls: labels},
		"server_collections":  {txt: "Number of collections loaded in game.", lbls: labels},
		"server_network_in":   {txt: "Ingress networking traffic.", lbls: labels},
		"server_network_out":  {txt: "Egress networking traffic.", lbls: labels},
		"server_restarting":   {txt: "1 if the server is restarting, 0 for running.", lbls: labels},
		"up":                  {txt: "Information about the FacepunchRust client", lbls: labels},
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

	e.extractServerInfo(ch)

	return nil
}

func (e *Exporter) extractServerInfo(ch chan<- prometheus.Metric) {
	log.Debugf("extractServerInfo()")
	serverInfoOutput, err := e.doRustCmd("serverinfo")
	if err != nil {
		log.Errorf("extractServerInfo() err: %s", err)
		return
	}
	si := ServerInfo{}
	json.Unmarshal([]byte(serverInfoOutput), &si)
	labels := e.getBuildInfoLabels()
	e.registerConstMetricGauge(ch, "players", float64(si.Players), labels...)
	e.registerConstMetricGauge(ch, "players_queued", float64(si.Queued), labels...)
	e.registerConstMetricGauge(ch, "players_joining", float64(si.Joining), labels...)
	e.registerConstMetricGauge(ch, "server_max_players", float64(si.MaxPlayers), labels...)
	e.registerConstMetricGauge(ch, "server_entity_count", float64(si.EntityCount), labels...)
	e.registerConstMetricGauge(ch, "server_uptime", float64(si.Uptime), labels...)
	e.registerConstMetricGauge(ch, "server_framerate", si.Framerate, labels...)
	e.registerConstMetricGauge(ch, "server_memory", float64(si.Memory), labels...)
	e.registerConstMetricGauge(ch, "server_collections", float64(si.Collections), labels...)
	e.registerConstMetricGauge(ch, "server_network_in", float64(si.NetworkIn), labels...)
	e.registerConstMetricGauge(ch, "server_network_out", float64(si.NetworkOut), labels...)
	e.registerConstMetricGauge(ch, "server_restarting", float64(func() int {
		if si.Restarting {
			return 1
		} else {
			return 0
		}
	}()), labels...)
}

func (e *Exporter) getBuildInfoLabels() []string {
	log.Debugf("extractBuildInfo()")
	buildInfoOutput, err := e.doRustCmd("buildinfo")
	if err != nil {
		log.Errorf("extractBuildInfo() err: %s", err)
		return []string{}
	}
	bi := ServerBuildInfo{}
	json.Unmarshal([]byte(buildInfoOutput), &bi)
	return []string{
		time.Unix(int64(bi.Date), 0).String(), bi.Build.Id, bi.Build.Number, bi.Build.Tag, bi.Build.Url, bi.Build.Name, bi.Build.Node, func() string {
			if bi.Valid {
				return "true"
			} else {
				return "false"
			}
		}(), bi.Scm.Type, bi.Scm.ChangeId, bi.Scm.Branch, bi.Scm.Repo, bi.Scm.Comment, bi.Scm.Author, bi.Scm.Date,
	}
}
