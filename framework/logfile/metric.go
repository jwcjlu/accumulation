package logfile

import (
	"context"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	LogMetricIndex = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cgvmagent",
		Subsystem: "logfile",
		Name:      "log_size",
		Help:      "log upload size MB",
		Buckets:   []float64{1, 5, 10, 15, 20, 25, 30, 35, 40, 45},
	}, []string{"area_type", "gid", "vmid", "code"})
)

func ReportLogMetric(ctx context.Context, code int, logSize float64) {
	obj := ctx.Value(_logMetricKey)
	objM, ok := obj.(*LogMetric)
	if ok {
		LogMetricIndex.WithLabelValues(objM.areaType, objM.gid, objM.vmid, strconv.Itoa(code)).Observe(logSize / 1024 / 1024)
	}
}

var _logMetricKey = "log_metric_key"

type LogMetric struct {
	areaType string
	gid      string
	vmid     string
}

func WithLogMetricContext(ctx context.Context, areaType, gid, vmid string) context.Context {
	return context.WithValue(ctx, _logMetricKey, &LogMetric{
		areaType: areaType,
		gid:      gid,
		vmid:     vmid,
	})
}

const (
	DoMoveFailureCode = 1
	UploadFailureCode = 2
	LogSizeExceed     = 3
	Success           = 0
)
