package service_catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// LivePodMetric is a single pod's live resource usage from Prometheus.
type LivePodMetric struct {
	Pod      string  `json:"pod"`
	CPUCores float64 `json:"cpu_cores"`
	MemoryGB float64 `json:"memory_gb"`
}

// PrometheusRepository abstracts fetching live metrics from a Prometheus instance.
type PrometheusRepository interface {
	GetPodMetrics(ctx context.Context, baseURL, namespace, service string) ([]LivePodMetric, error)
}

type httpPrometheusRepository struct {
	client *http.Client
	logger *zap.Logger
}

func NewHTTPPrometheusRepository(logger *zap.Logger) PrometheusRepository {
	return &httpPrometheusRepository{
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

func (r *httpPrometheusRepository) GetPodMetrics(ctx context.Context, baseURL, namespace, service string) ([]LivePodMetric, error) {
	cpuQuery := fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod=~"%s.*",container!=""}[5m])) by (pod)`,
		namespace, service,
	)
	memQuery := fmt.Sprintf(
		`sum(container_memory_working_set_bytes{namespace="%s",pod=~"%s.*",container!=""}) by (pod)`,
		namespace, service,
	)

	cpuVecs, err := r.query(ctx, baseURL, cpuQuery)
	if err != nil {
		return nil, fmt.Errorf("cpu query: %w", err)
	}
	memVecs, err := r.query(ctx, baseURL, memQuery)
	if err != nil {
		return nil, fmt.Errorf("memory query: %w", err)
	}

	cpuByPod := make(map[string]float64, len(cpuVecs))
	for _, v := range cpuVecs {
		cpuByPod[v.Metric["pod"]] = promScalar(v.Value)
	}
	memByPod := make(map[string]float64, len(memVecs))
	for _, v := range memVecs {
		memByPod[v.Metric["pod"]] = promScalar(v.Value) / (1024 * 1024 * 1024)
	}

	seen := make(map[string]struct{}, len(cpuByPod)+len(memByPod))
	for p := range cpuByPod {
		seen[p] = struct{}{}
	}
	for p := range memByPod {
		seen[p] = struct{}{}
	}

	pods := make([]LivePodMetric, 0, len(seen))
	for pod := range seen {
		pods = append(pods, LivePodMetric{
			Pod:      pod,
			CPUCores: cpuByPod[pod],
			MemoryGB: memByPod[pod],
		})
	}
	r.logger.Debug("prometheus pod metrics fetched",
		zap.String("namespace", namespace), zap.String("service", service), zap.Int("pods", len(pods)))
	return pods, nil
}

// ── internal ──────────────────────────────────────────────────────────────────

type promResponse struct {
	Status string       `json:"status"`
	Data   promDataBody `json:"data"`
}
type promDataBody struct {
	Result []promVector `json:"result"`
}
type promVector struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"` // [unix_ts_float, value_string]
}

func (r *httpPrometheusRepository) query(ctx context.Context, baseURL, query string) ([]promVector, error) {
	u := baseURL + "/api/v1/query?" + url.Values{"query": {query}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("unmarshal prometheus response: %w", err)
	}
	if pr.Status != "success" {
		return nil, fmt.Errorf("prometheus returned status %q", pr.Status)
	}
	return pr.Data.Result, nil
}

func promScalar(v []interface{}) float64 {
	if len(v) < 2 {
		return 0
	}
	s, _ := v[1].(string)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
