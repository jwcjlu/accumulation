package api

import "context"

type BandwidthReportClient interface {
	ReportBandwidthData(ctx context.Context, data interface{}) error
}
