package report

import (
	"accumulation/bandwidth/api"
	"accumulation/bandwidth/conf"
	"accumulation/bandwidth/model"
	"accumulation/bandwidth/store"
	"accumulation/pkg/log"
	"context"

	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var BufLen = 500

type BandwidthReportJob struct {
	client       api.BandwidthReportClient
	isRunnable   atomic.Bool
	ringBuff     *RingBuff[*model.ReportFlowBizRequest]
	tmp          []*model.ReportFlowBizRequest
	mutex        *sync.Mutex
	dataBuffOnly atomic.Bool //数据仅仅在buff里
	store        *store.FileStore[*model.ReportFlowBizRequest]
}

func NewBandwidthReportJob(
	client api.BandwidthReportClient,
	config *conf.Data,
) *BandwidthReportJob {
	if config.Acl.ReportConfig != nil && config.Acl.ReportConfig.ReportJobBufLen > 0 {
		BufLen = int(config.Acl.ReportConfig.ReportJobBufLen)
	}
	return &BandwidthReportJob{
		client:   client,
		mutex:    &sync.Mutex{},
		ringBuff: NewRingBuff[*model.ReportFlowBizRequest](BufLen),
		store:    store.NewFileStore[*model.ReportFlowBizRequest](""),
	}
}

func (job *BandwidthReportJob) Start(ctx context.Context) error {
	job.isRunnable.Swap(true)
	err := job.store.Open()
	if err != nil {
		return err
	}
	reqs, err := job.loadFromFile(ctx, BufLen)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		job.Add(ctx, req, true)
	}
	if job.ringBuff.SurplusCount() > 0 {
		job.dataBuffOnly.Store(true)
	}
	go job.loopReport()
	return nil
}
func (job *BandwidthReportJob) Stop(ctx context.Context) error {
	job.isRunnable.Swap(false)
	surplus := job.ringBuff.Surplus()
	job.store.Truncate(ctx, surplus, job.tmp)
	return job.store.Close()
}
func (job *BandwidthReportJob) loopReport() {
	defer func() {
		if e := recover(); e != nil {
			log.Errorf(context.TODO(), "BandwidthReportJob panic|err=%v|stack=%v", e, string(debug.Stack()))
		}
		if job.isRunnable.Load() {
			job.loopReport()
		}
	}()
	for job.isRunnable.Load() {
		req := job.ringBuff.Current()
		if req != nil {
			err := job.client.ReportBandwidthData(context.Background(), []*model.ReportFlowBizRequest{req})
			if err != nil {
				time.Sleep(time.Second * time.Duration(2))
				log.Warnf(context.Background(), "report bandwidth failure req:%v", *req)
			} else {
				job.ringBuff.Dequeue()
			}
		} else {
			job.tryLoadData()
			time.Sleep(time.Second * time.Duration(3))
		}
	}
}
func (job *BandwidthReportJob) Add(ctx context.Context, req *model.ReportFlowBizRequest, forceAdd bool) {
	job.mutex.Lock()
	defer job.mutex.Unlock()
	job.doAdd(ctx, req, forceAdd)
}

func (job *BandwidthReportJob) doAdd(ctx context.Context, req *model.ReportFlowBizRequest, forceAdd bool) {
	//数据仅仅在buff里，才会试图进入到环形队列里面，否则就写到缓存里面
	if (job.dataBuffOnly.Load() || forceAdd) && job.ringBuff.Enqueue(req) {
		return
	}
	job.tmp = append(job.tmp, req)
	if len(job.tmp) >= BufLen { //大于缓存就的写到文件里面
		err := job.writeToFile(ctx, job.tmp)
		if err != nil {
			log.Warnf(context.Background(), "loadToFile failure err:%v", err)
		} else {
			job.tmp = nil
		}
	}
	job.dataBuffOnly.Store(false)
}

// 试着加载数据
// 1、先加载文件里面的数据，如果文件的数据加载完了再tmp的数据
// 2、加载完tmp的数据到环形队列里面，如果tmp加载完了的话，说明只有buff里面有数据,dataBuffOnly改成true
func (job *BandwidthReportJob) tryLoadData() {
	job.mutex.Lock()
	defer job.mutex.Unlock()
	if job.ringBuff.SurplusCount() == 0 {
		return
	}
	reportFlowBizRequests, err := job.loadFromFile(context.TODO(), job.ringBuff.SurplusCount())
	if err != nil {
		log.Errorf(context.TODO(), "loadFromFile failure err:%v", err)
	}
	for _, reportFlowBizRequest := range reportFlowBizRequests {
		job.doAdd(context.TODO(), reportFlowBizRequest, true)
	}
	index := 0
	for _, t := range job.tmp {
		if job.ringBuff.SurplusCount() > 0 {
			job.doAdd(context.TODO(), t, true)
		} else {
			job.tmp = job.tmp[index:]
			return
		}
		index++
	}
	job.tmp = nil
	job.dataBuffOnly.Store(true)

}
func (job *BandwidthReportJob) writeToFile(ctx context.Context, data []*model.ReportFlowBizRequest) error {
	return job.store.Store(ctx, data)
}
func (job *BandwidthReportJob) loadFromFile(ctx context.Context, rows int) ([]*model.ReportFlowBizRequest, error) {
	return job.store.Load(ctx, rows)
}
