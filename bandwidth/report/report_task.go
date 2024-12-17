package report

import (
	"accumulation/bandwidth/api"
	"accumulation/bandwidth/model"
	"accumulation/bandwidth/store"
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"

	"runtime/debug"
	"sync/atomic"
	"time"
)

type BandwidthReportTask struct {
	session        *model.Session
	isRunnable     atomic.Bool
	lastStatTime   int64
	engine         store.BandwidthEngine
	cron           *cron.Cron
	manager        api.BandwidthReportManager
	sessionTimeout int64
	no             int32
	job            *BandwidthReportJob
	hardwareType   string
}

func NewBandwidthReportTask(
	sess *model.Session,
	engine store.BandwidthEngine,
	job *BandwidthReportJob,
	manager api.BandwidthReportManager) *BandwidthReportTask {
	return &BandwidthReportTask{session: sess,
		engine:       engine,
		manager:      manager,
		job:          job,
		hardwareType: fmt.Sprintf("%s%s", "", ""),
	}
}

func (trt *BandwidthReportTask) Stop(ctx context.Context) {
	trt.cron.Stop()
	trt.isRunnable.Swap(false)
	log.Infof("flowId %s,max no %d", trt.session.FlowID, trt.no)
	trt.manager.RemoveTask(context.Background(), trt.session)
}

func (trt *BandwidthReportTask) Start(ctx context.Context) {
	log.Infof("session[%s]  start collect bandwidth",
		trt.session.String())
	trt.isRunnable.Swap(true)
	trt.lastStatTime = trt.session.Start
	trt.backendReport()

}

func (trt *BandwidthReportTask) backendReport() {
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("backendReport panic|err=%v|stack=%v", e, string(debug.Stack()))
		}
	}()

	trt.cron = cron.New(cron.WithSeconds())
	_, err := trt.cron.AddFunc("*/10 * * * * *", func() {
		trt.periodFetchBandwidth()
	})
	trt.cron.Start()
	if err != nil {
		log.Errorf("c.AddFunc error|err=%v|stack=%v", err)
	}

}
func (trt *BandwidthReportTask) periodFetchBandwidth() {
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("execReport panic|err=%v|stack=%v", e, string(debug.Stack()))
		}
	}()
	now := time.Now().Unix()
	filter := func(bandwidth *model.Bandwidth) bool {
		if len(trt.session.StreamIp) > 0 && bandwidth.Ip != trt.session.StreamIp {
			return false
		}
		return true
	}
	bandwidths, err := trt.engine.Query(context.TODO(), filter, trt.lastStatTime, now)
	if err != nil {
		log.Errorf("query traffic failure err:%v", err)
		return
	}
	if len(bandwidths) == 0 {
		return
	}
	trt.lastStatTime = now
	upTotal, downTotal := model.NewBandwidths(bandwidths).Group()
	if upTotal+downTotal < 1 {
		return
	}
	portFilter := func(bandwidth *model.Bandwidth) bool {
		if len(trt.session.StreamPorts) > 0 && !trt.session.StreamPorts.Contains(bandwidth.Port) {
			return false
		}
		return true
	}
	upstream, downstream := model.NewBandwidths(model.NewBandwidths(bandwidths).Filter(portFilter)).Group()
	if upstream+downstream < 1 {
		return
	}
	trt.job.Add(context.TODO(), trt.buildReportFlowBizRequest(upTotal, downTotal, upstream, downstream), false)
	trt.no++
}

func (trt *BandwidthReportTask) buildReportFlowBizRequest(upTotal, downTotal, upstream, downstream int32) *model.ReportFlowBizRequest {

	return &model.ReportFlowBizRequest{}

}

func (trt *BandwidthReportTask) IsRunnable() bool {
	return trt.isRunnable.Load()
}
