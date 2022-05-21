package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/jamesalbert/facepunch_rust_exporter/v2/exporter"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var (
	BuildVersion   = ""
	BuildDate      = ""
	BuildCommitSha = ""
)

func getEnv(key string, defaultVal string) string {
	if envVal, ok := os.LookupEnv(key); ok {
		return envVal
	}
	return defaultVal
}

func main() {
	var (
		rustAddr      = flag.String("rust.addr", getEnv("RUST_ADDR", ""), "Address of the Rust server to scrape")
		rustPwd       = flag.String("rcon.password", getEnv("RCON_PASSWORD", ""), "Password of the Rust server rcon")
		listenAddress = flag.String("web.listen-address", getEnv("EXPORTER_WEB_LISTEN_ADDRESS", ":1337"), "Address to listen on for web interface and telemetry.")
		metricPath    = flag.String("web.telemetry-path", getEnv("EXPORTER_WEB_TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
		namespace     = "facepunch_rust"
	)

	flag.Parse()

	registry := prometheus.NewRegistry()

	exp, err := exporter.NewFacepunchRustExporter(*rustAddr, exporter.Options{
		Password:    *rustPwd,
		MetricsPath: *metricPath,
		Namespace:   namespace,
		Registry:    registry,
		BuildInfo: exporter.BuildInfo{
			Version:   BuildVersion,
			CommitSha: BuildCommitSha,
			Date:      BuildDate,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Providing metrics at %s%s", *listenAddress, *metricPath)
	log.Debugf("Configured rust addr: %#v", *rustAddr)
	log.Fatal(http.ListenAndServe(*listenAddress, exp))
}
