package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

const defaultServeAddr = "127.0.0.1:9464"

var serveAddr = defaultServeAddr

var serveListenAndServe = func(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run a localhost observability HTTP server",
	Long: `Run a localhost observability HTTP server that provides health, status, metrics, and data APIs.

Available API endpoints:
  /healthz        - Health check
  /status         - Current configuration and sync status
  /metrics        - Prometheus metrics
  /api/summary    - Latest health metrics summary
  /api/recovery   - Daily recovery data
  /api/sleep      - Daily sleep data
  /api/strain     - Daily strain data
  /api/workouts   - Workout summary data

Query parameters for /api endpoints:
  from            - Start date (YYYY-MM-DD)
  to              - End date (YYYY-MM-DD), inclusive
  limit           - Number of records to return (default: 90, max: 1000)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		handler := newServeHandler(buildServeStatusReport)
		fmt.Fprintf(cmd.OutOrStdout(), "Listening on http://%s\n", serveAddr)
		err := serveListenAndServe(serveAddr, handler)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	},
}

type statusReporter func() statusReport

func buildServeStatusReport() statusReport {
	return buildStatusReportWithOpenDB(store.OpenReadOnly)
}

func newServeHandler(reporter statusReporter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/status", statusHandler(reporter))
	mux.HandleFunc("/api/summary", apiSummaryHandler(reporter))
	mux.HandleFunc("/api/recovery", apiRowsHandler("daily_recovery", "day", "day"))
	mux.HandleFunc("/api/sleep", apiRowsHandler("daily_sleep", "day", "day"))
	mux.HandleFunc("/api/strain", apiRowsHandler("daily_strain", "day", "day"))
	mux.HandleFunc("/api/workouts", apiRowsHandler("workout_summary", "day", "start"))

	registry := prometheus.NewRegistry()
	registry.MustRegister(newStatusCollector(reporter))
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}

func statusHandler(reporter statusReporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = writeStatusJSON(w, reporter())
	}
}

type statusCollector struct {
	reporter statusReporter

	dbOpen                 *prometheus.Desc
	tokenPresent           *prometheus.Desc
	clientIDConfigured     *prometheus.Desc
	clientSecretConfigured *prometheus.Desc
	recordsTotal           *prometheus.Desc
	lastSyncTimestamp      *prometheus.Desc
	syncAge                *prometheus.Desc
	syncStale              *prometheus.Desc
	statusErrorsTotal      *prometheus.Desc
	latestHealthMetric     *prometheus.Desc
	latestHealthTimestamp  *prometheus.Desc
	alertsEnabled          *prometheus.Desc
	alertThreshold         *prometheus.Desc
	alertsFiring           *prometheus.Desc
	alertState             *prometheus.Desc
}

func newStatusCollector(reporter statusReporter) prometheus.Collector {
	return &statusCollector{
		reporter: reporter,
		dbOpen: prometheus.NewDesc(
			"whooper_db_open",
			"Whether the Whooper SQLite database opens successfully.",
			nil,
			nil,
		),
		tokenPresent: prometheus.NewDesc(
			"whooper_token_present",
			"Whether a Whooper OAuth token is present.",
			nil,
			nil,
		),
		clientIDConfigured: prometheus.NewDesc(
			"whooper_client_id_configured",
			"Whether a Whooper client ID is configured.",
			nil,
			nil,
		),
		clientSecretConfigured: prometheus.NewDesc(
			"whooper_client_secret_configured",
			"Whether a Whooper client secret is configured.",
			nil,
			nil,
		),
		recordsTotal: prometheus.NewDesc(
			"whooper_records_total",
			"Number of locally stored Whooper records by entity.",
			[]string{"entity"},
			nil,
		),
		lastSyncTimestamp: prometheus.NewDesc(
			"whooper_last_sync_timestamp_seconds",
			"Last successful persisted sync timestamp by entity, or 0 if never synced or unparsable.",
			[]string{"entity"},
			nil,
		),
		syncAge: prometheus.NewDesc(
			"whooper_sync_age_seconds",
			"Seconds since last successful local sync timestamp for entity; 0 if unknown.",
			[]string{"entity"},
			nil,
		),
		syncStale: prometheus.NewDesc(
			"whooper_sync_stale",
			"1 when sync age is older than 24h, otherwise 0; 0 if unknown.",
			[]string{"entity"},
			nil,
		),
		statusErrorsTotal: prometheus.NewDesc(
			"whooper_status_errors_total",
			"Count of current status-report errors.",
			nil,
			nil,
		),
		latestHealthMetric: prometheus.NewDesc(
			"whooper_latest_health_metric",
			"Latest locally cached WHOOP health metric value.",
			[]string{"metric"},
			nil,
		),
		latestHealthTimestamp: prometheus.NewDesc(
			"whooper_latest_health_timestamp_seconds",
			"Timestamp for the latest locally cached WHOOP health metric group.",
			[]string{"entity"},
			nil,
		),
		alertsEnabled: prometheus.NewDesc(
			"whooper_alerts_enabled",
			"Whether health alerts are enabled.",
			nil,
			nil,
		),
		alertThreshold: prometheus.NewDesc(
			"whooper_alert_threshold",
			"Configured alert threshold values.",
			[]string{"type"},
			nil,
		),
		alertsFiring: prometheus.NewDesc(
			"whooper_alerts_firing",
			"Number of currently firing alerts.",
			nil,
			nil,
		),
		alertState: prometheus.NewDesc(
			"whooper_alert_state",
			"Current firing state of specific alerts (1 for firing, 0 otherwise).",
			[]string{"alert"},
			nil,
		),
	}
}

func (c *statusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.dbOpen
	ch <- c.tokenPresent
	ch <- c.clientIDConfigured
	ch <- c.clientSecretConfigured
	ch <- c.recordsTotal
	ch <- c.lastSyncTimestamp
	ch <- c.syncAge
	ch <- c.syncStale
	ch <- c.statusErrorsTotal
	ch <- c.latestHealthMetric
	ch <- c.latestHealthTimestamp
	ch <- c.alertsEnabled
	ch <- c.alertThreshold
	ch <- c.alertsFiring
	ch <- c.alertState
}

func (c *statusCollector) Collect(ch chan<- prometheus.Metric) {
	report := c.reporter()
	ch <- prometheus.MustNewConstMetric(c.dbOpen, prometheus.GaugeValue, boolGauge(report.DBOpen))
	ch <- prometheus.MustNewConstMetric(c.tokenPresent, prometheus.GaugeValue, boolGauge(report.TokenPresent))
	ch <- prometheus.MustNewConstMetric(c.clientIDConfigured, prometheus.GaugeValue, boolGauge(report.ClientIDConfigured))
	ch <- prometheus.MustNewConstMetric(c.clientSecretConfigured, prometheus.GaugeValue, boolGauge(report.ClientSecretConfigured))

	now := time.Now()
	for _, entity := range statusEntities() {
		ch <- prometheus.MustNewConstMetric(c.recordsTotal, prometheus.GaugeValue, float64(report.RecordCounts[entity]), entity)
		ts := syncTimestampSeconds(report.LastSync[entity])
		ch <- prometheus.MustNewConstMetric(c.lastSyncTimestamp, prometheus.GaugeValue, ts, entity)

		age := syncAgeSeconds(report.LastSync[entity], now)
		ch <- prometheus.MustNewConstMetric(c.syncAge, prometheus.GaugeValue, age, entity)
		ch <- prometheus.MustNewConstMetric(c.syncStale, prometheus.GaugeValue, syncStaleGauge(age), entity)
	}

	ch <- prometheus.MustNewConstMetric(c.statusErrorsTotal, prometheus.GaugeValue, float64(len(report.Errors)))

	if report.LatestHealth != nil {
		metrics := make([]string, 0, len(report.LatestHealth.Values))
		for metric := range report.LatestHealth.Values {
			metrics = append(metrics, metric)
		}
		sort.Strings(metrics)
		for _, metric := range metrics {
			ch <- prometheus.MustNewConstMetric(c.latestHealthMetric, prometheus.GaugeValue, report.LatestHealth.Values[metric], metric)
		}

		entities := make([]string, 0, len(report.LatestHealth.Timestamps))
		for entity := range report.LatestHealth.Timestamps {
			entities = append(entities, entity)
		}
		sort.Strings(entities)
		for _, entity := range entities {
			ch <- prometheus.MustNewConstMetric(c.latestHealthTimestamp, prometheus.GaugeValue, healthTimestampSeconds(report.LatestHealth.Timestamps[entity]), entity)
		}
	}

	ch <- prometheus.MustNewConstMetric(c.alertsEnabled, prometheus.GaugeValue, boolGauge(report.AlertsEnabled))
	ch <- prometheus.MustNewConstMetric(c.alertThreshold, prometheus.GaugeValue, report.LowRecoveryThreshold, "low_recovery")
	ch <- prometheus.MustNewConstMetric(c.alertThreshold, prometheus.GaugeValue, report.HighStrainThreshold, "high_strain")
	ch <- prometheus.MustNewConstMetric(c.alertsFiring, prometheus.GaugeValue, float64(report.AlertsFiring))

	// Ensure stable output for standard alerts
	standardAlerts := []string{"low_recovery", "high_strain"}
	for _, name := range standardAlerts {
		val := 0
		if report.AlertStates != nil {
			val = report.AlertStates[name]
		}
		ch <- prometheus.MustNewConstMetric(c.alertState, prometheus.GaugeValue, float64(val), name)
	}

	// Emit any other custom alerts
	var otherAlerts []string
	for name := range report.AlertStates {
		isStandard := false
		for _, s := range standardAlerts {
			if name == s {
				isStandard = true
				break
			}
		}
		if !isStandard {
			otherAlerts = append(otherAlerts, name)
		}
	}
	sort.Strings(otherAlerts)
	for _, name := range otherAlerts {
		ch <- prometheus.MustNewConstMetric(c.alertState, prometheus.GaugeValue, float64(report.AlertStates[name]), name)
	}
}

func boolGauge(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func syncTimestampSeconds(value string) float64 {
	if value == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0
	}
	return float64(t.Unix())
}

func syncAgeSeconds(value string, now time.Time) float64 {
	if value == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0
	}
	age := now.Sub(t).Seconds()
	if age < 0 {
		return 0
	}
	return age
}

func syncStaleGauge(ageSeconds float64) float64 {
	if ageSeconds > (24 * time.Hour).Seconds() {
		return 1
	}
	return 0
}

func healthTimestampSeconds(value string) float64 {
	if value == "" {
		return 0
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return float64(t.Unix())
	}
	return syncTimestampSeconds(value)
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", defaultServeAddr, "Address to bind the observability server")
	rootCmd.AddCommand(serveCmd)
}
