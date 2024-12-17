package store

import (
	"accumulation/bandwidth/collector"
	"accumulation/bandwidth/model"
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/robfig/cron/v3"
)

type BandwidthEngine interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Store(ctx context.Context, traffics []*model.Bandwidth) error
	Query(ctx context.Context, filter model.Filters, startTime, endTime int64) ([]*model.Bandwidth, error)
}

type RejectPolicy interface {
	Handle(traffics []*model.Bandwidth)
}

type fileEngine struct {
	data       model.Bandwidths
	collectors []*collector.BandwidthCollector
	cron       *cron.Cron
	mutex      *sync.RWMutex
	capacity   int
}

func (t *fileEngine) Start(ctx context.Context) error {
	t.backendCollector()
	return nil

}

func (t *fileEngine) Stop(ctx context.Context) error {
	t.cron.Stop()
	t.data = nil
	return nil
}

func (t *fileEngine) Query(ctx context.Context, filter model.Filters, startTime, endTime int64) ([]*model.Bandwidth, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	data := t.data.Search(startTime, endTime)
	if len(data) == 0 {
		return nil, nil
	}
	return model.NewBandwidths(data).Filter(filter), nil
}

func (t *fileEngine) backendCollector() {
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("backendReport panic|err=%v|stack=%v", e, string(debug.Stack()))
		}
	}()
	t.cron = cron.New(cron.WithSeconds())
	_, err := t.cron.AddFunc("*/5 * * * * *", func() {
		t.collector()
	})
	t.cron.Start()
	if err != nil {
		log.Errorf("c.AddFunc error|err=%v|stack=%v", err)
	}

}

func (t *fileEngine) collector() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	var bandwidths []*model.Bandwidth
	for _, collector := range t.collectors {
		bandwidths = append(bandwidths, collector.ExportAndClean()...)
	}
	if len(bandwidths) > 0 {
		t.Add(bandwidths)
	}
}

func (t *fileEngine) Add(bandwidths []*model.Bandwidth) {
	defer t.data.Sort()
	length := len(bandwidths)
	if length > t.capacity {
		t.data = bandwidths[length-t.capacity:]
		return
	}
	reduce := length + len(t.data) - t.capacity
	if reduce > 0 {
		if reduce < t.capacity/2 {
			reduce = t.capacity / 2
		}
		t.data = t.data[reduce:]
	}
	t.data = append(t.data, bandwidths...)
}
func (t *fileEngine) Store(ctx context.Context, bandwidths []*model.Bandwidth) error {
	return fmt.Errorf("not implement")
}
func NewBandwidthEngine(collectors []*collector.BandwidthCollector, capacity int) BandwidthEngine {
	if capacity > 0 {
		defaultCapacity = capacity
	}
	return &fileEngine{collectors: collectors, mutex: &sync.RWMutex{}, capacity: defaultCapacity}
}

var defaultCapacity = 4000
