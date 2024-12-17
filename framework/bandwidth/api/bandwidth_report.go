package api

import (
	"accumulation/framework/bandwidth/model"
	"context"
)

type BandwidthReportManager interface {
	StartReport(ctx context.Context, session *model.Session) error

	EndReport(ctx context.Context, session *model.Session) error

	RemoveTask(ctx context.Context, session *model.Session) error

	NotifyAccessInfo(ctx context.Context, vmid int64, streamIp string, streamPort model.StreamPorts) error
}
