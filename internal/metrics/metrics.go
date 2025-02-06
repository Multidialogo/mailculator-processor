package metrics

import (
	"log"
	"net/http"
	"mailculator-processor/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define the metrics
var ProcessedFilesCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "app_processed_files_total",
		Help: "Total number of files processed.",
	},
	[]string{"status"},
)

var InProgressFilesGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "app_in_progress_files",
		Help: "Number of files being processed.",
	},
	[]string{"endpoint"},
)

var MemoryUsageGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "app_memory_usage_bytes",
		Help: "Amount of memory used by the application.",
	},
	[]string{"type"},
)

var CpuUsageGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "app_cpu_usage_percent",
		Help: "CPU usage percentage.",
	},
	[]string{"cpu"},
)

// Init initializes and registers the Prometheus metrics and starts the HTTP server to expose them
func Init(envName string) {
	// Retrieve paths from the configuration
	registry := config.GetRegistry()

	// Register metrics with Prometheus
	prometheus.MustRegister(ProcessedFilesCounter)
	prometheus.MustRegister(InProgressFilesGauge)
	prometheus.MustRegister(MemoryUsageGauge)
	prometheus.MustRegister(CpuUsageGauge)

	// Start the HTTP server in a goroutine to expose metrics at /metrics endpoint
	if envName == "PROD" || envName == "DEV" {
		prometheusPort := registry.Get("PROMETHEUS_PORT")
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Println("Starting Prometheus metrics server on :%d", prometheusPort)
			if err := http.ListenAndServe(":"+prometheusPort, nil); err != nil {
				log.Fatalf("Error starting Prometheus server: %v", err)
			}
		}()
	}
}
