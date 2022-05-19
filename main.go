package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/jamesalbert/facepunch_rust_exporter/v2/exporter"
	log "github.com/sirupsen/logrus"
)

func getEnv(key string, defaultVal string) string {
	if envVal, ok := os.LookupEnv(key); ok {
		return envVal
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if envVal, ok := os.LookupEnv(key); ok {
		envBool, err := strconv.ParseBool(envVal)
		if err == nil {
			return envBool
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if envVal, ok := os.LookupEnv(key); ok {
		envInt64, err := strconv.ParseInt(envVal, 10, 64)
		if err == nil {
			return envInt64
		}
	}
	return defaultVal
}

func main() {
	var (
		rustAddr      = flag.String("rust.addr", getEnv("RUST_ADDR", "localhost:28016"), "Address of the Redis instance to scrape")
		rustPwd       = flag.String("rust.password", getEnv("RUST_PASSWORD", ""), "Password of the Redis instance to scrape")
		listenAddress = flag.String("web.listen-address", getEnv("CHEF_EXPORTER_WEB_LISTEN_ADDRESS", ":9121"), "Address to listen on for web interface and telemetry.")
		metricPath    = flag.String("web.telemetry-path", getEnv("CHEF_EXPORTER_WEB_TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
	)

	flag.Parse()

	exp, err := exporter.NewFacepunchRustExporter(*rustAddr, exporter.Options{
		Password:    *rustPwd,
		MetricsPath: *metricPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Providing metrics at %s%s", *listenAddress, *metricPath)
	log.Debugf("Configured rust addr: %#v", *rustAddr)
	log.Fatal(http.ListenAndServe(*listenAddress, exp))
}
