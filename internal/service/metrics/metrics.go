package metrics

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type Metrics struct {
	ProcessedFilesCounter *prometheus.CounterVec
	InProgressFilesGauge  *prometheus.GaugeVec
	MemoryUsageGauge      *prometheus.GaugeVec
	CpuUsageGauge         *prometheus.GaugeVec
}

// Init initializes and registers the Prometheus metrics and starts the HTTP server to expose them
func NewMetrics(startHttpServer bool, httpPort int) *Metrics {
	metrics := &Metrics{
		ProcessedFilesCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "app_processed_files_total",
				Help: "Total number of files processed.",
			},
			[]string{"status"},
		),
		InProgressFilesGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_in_progress_files",
				Help: "Number of files being processed.",
			},
			[]string{"endpoint"},
		),
		MemoryUsageGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_memory_usage_bytes",
				Help: "Amount of memory used by the application.",
			},
			[]string{"type"},
		),
		CpuUsageGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "app_cpu_usage_percent",
				Help: "CPU usage percentage.",
			},
			[]string{"cpu"},
		),
	}

	// Register metrics with Prometheus
	prometheus.MustRegister(metrics.ProcessedFilesCounter)
	prometheus.MustRegister(metrics.InProgressFilesGauge)
	prometheus.MustRegister(metrics.MemoryUsageGauge)
	prometheus.MustRegister(metrics.CpuUsageGauge)

	// Start the HTTP server in a goroutine to expose metrics at /metrics endpoint
	if startHttpServer {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			log.Printf("\u001B[34mINFO: Starting Prometheus metrics server on :%d\u001B[0m", httpPort)
			if err := http.ListenAndServe(":"+strconv.Itoa(httpPort), nil); err != nil {
				log.Fatalf("Error starting Prometheus server: %v", err)
			}
		}()
	}

	return metrics
}

func (m *Metrics) CollectMemoryAndCpu() error {

	// Memory stat
	memory, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("Error fetching memory usage: %v", err)
	}

	// CPU stats
	cpus, err := cpu.Percent(0, false)
	if err != nil {
		return fmt.Errorf("Error fetching memory usage: %v", err)
	}

	m.MemoryUsageGauge.WithLabelValues("total").Set(float64(memory.Total)) // Total memory in bytes
	m.MemoryUsageGauge.WithLabelValues("used").Set(float64(memory.Used))   // Used memory in bytes
	m.MemoryUsageGauge.WithLabelValues("free").Set(float64(memory.Free))   // Free memory in bytes
	m.MemoryUsageGauge.WithLabelValues("percent").Set(memory.UsedPercent)  // Percent used

	for i, cpuUsage := range cpus {
		label := fmt.Sprintf("cpu%d", i)
		m.CpuUsageGauge.WithLabelValues(label).Set(cpuUsage)
	}

	return nil
}
