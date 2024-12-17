package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "handler", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0},
		},
		[]string{"handler"},
	)
	notCarryVmidFromHeader = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "not_carry_vmid_req",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "handler", "clientIp"},
	)
	otherRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_other_request_duration_seconds",
			Help:    "HTTP other request duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0},
		},
		[]string{"handler", "code"},
	)
)

func init() {
	// 注册指标到 Prometheus 默认注册表
	prometheus.MustRegister(requests)
	prometheus.MustRegister(notCarryVmidFromHeader)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(otherRequestDuration)
}

// PrometheusMiddleware 监控中间件
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler := mux.CurrentRoute(r).GetName() // 获取当前路由的名称
		if len(handler) == 0 {
			handler = r.RequestURI
		}
		// 包装 ResponseWriter 以捕获状态码
		pw := &promResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(pw, r)

		// 记录请求指标
		requests.WithLabelValues(r.Method, handler, fmt.Sprint(pw.statusCode)).Inc()
		requestDuration.WithLabelValues(handler).Observe(time.Since(start).Seconds())
	})
}

// 自定义 ResponseWriter 以捕获状态码
type promResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (prw *promResponseWriter) WriteHeader(code int) {
	prw.statusCode = code
	prw.ResponseWriter.WriteHeader(code)
}

func ReportNotCarryVmidFromHeader(method, handler, clientIp string) {
	notCarryVmidFromHeader.WithLabelValues(method, handler, clientIp).Inc()
}
func ReportOtherRequestDuration(handler string, code int, start time.Time) {
	otherRequestDuration.WithLabelValues(handler, fmt.Sprintf("%d", code)).Observe(time.Since(start).Seconds())
}
