package exporter

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (e *Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mux.ServeHTTP(w, r)
}

func (e *Exporter) healthHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`ok`))
}

func (e *Exporter) indexHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte(`<html>
<head><title>FacepunchRust Exporter ` + e.buildInfo.Version + `</title></head>
<body>
<h1>FacepunchRust Exporter ` + e.buildInfo.Version + `</h1>
<p><a href='` + e.options.MetricsPath + `'>Metrics</a></p>
</body>
</html>
`))
}

func (e *Exporter) scrapeHandler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("target")
	if target == "" {
		message := "'target' parameter must be specified"
		log.Error(message)
		http.Error(w, message, http.StatusBadRequest)
		e.targetScrapeRequestErrors.Inc()
		return
	}

	opts := e.options

	registry := prometheus.NewRegistry()
	opts.Registry = registry

	_, err := NewFacepunchRustExporter(target, opts)
	if err != nil {
		message := "NewFacepunchRustExporter() err: err"
		log.Error(message)
		http.Error(w, message, http.StatusBadRequest)
		e.targetScrapeRequestErrors.Inc()
		return
	}

	promhttp.HandlerFor(
		registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},
	).ServeHTTP(w, r)
}
