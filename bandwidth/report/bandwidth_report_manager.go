package report

import (
	"accumulation/bandwidth/api"
	"accumulation/bandwidth/collector"
	"accumulation/bandwidth/conf"
	"accumulation/bandwidth/model"
	"accumulation/bandwidth/store"
	"accumulation/pkg/nnet"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/gopacket/pcap"
	"net"
	"sync"
)

const defaultBpfFilter = "udp or tcp"

type bandwidthReportManager struct {
	client       api.BandwidthReportClient
	collectors   []*collector.BandwidthCollector
	engine       store.BandwidthEngine
	mutex        *sync.Mutex
	tasks        map[string]*BandwidthReportTask
	reportConfig *conf.Acl_ReportConfig
	job          *BandwidthReportJob
}

func NewBandwidthReportManager(client api.BandwidthReportClient, job *BandwidthReportJob, data *conf.Data) api.BandwidthReportManager {
	return &bandwidthReportManager{
		client:       client,
		mutex:        &sync.Mutex{},
		reportConfig: data.Acl.ReportConfig,
		tasks:        make(map[string]*BandwidthReportTask),
		job:          job,
	}

}
func (bandwidthReportManager *bandwidthReportManager) StartReport(ctx context.Context, session *model.Session) error {
	task, ok := bandwidthReportManager.tasks[session.SessionKey()]
	if ok {
		task.Stop(ctx)
	}
	bandwidthReportManager.mutex.Lock()
	defer bandwidthReportManager.mutex.Unlock()
	if bandwidthReportManager.collectors == nil {
		err := bandwidthReportManager.initialization(ctx)
		if err != nil {
			return err
		}
	}
	task = NewBandwidthReportTask(session,
		bandwidthReportManager.engine,
		bandwidthReportManager.job,
		bandwidthReportManager)
	task.Start(ctx)
	bandwidthReportManager.tasks[session.SessionKey()] = task
	return nil
}

func (bandwidthReportManager *bandwidthReportManager) NotifyAccessInfo(ctx context.Context,
	vmid int64, streamIp string, streamPorts model.StreamPorts) error {

	return nil
}
func (bandwidthReportManager *bandwidthReportManager) RemoveTask(ctx context.Context, session *model.Session) error {
	bandwidthReportManager.mutex.Lock()
	defer bandwidthReportManager.mutex.Unlock()
	delete(bandwidthReportManager.tasks, session.SessionKey())
	if len(bandwidthReportManager.tasks) == 0 {
		for _, collector := range bandwidthReportManager.collectors {
			collector.Stop(ctx)
		}
		bandwidthReportManager.engine.Stop(ctx)
		bandwidthReportManager.collectors = nil
		bandwidthReportManager.engine = nil
	}
	return nil
}
func (bandwidthReportManager *bandwidthReportManager) EndReport(ctx context.Context, session *model.Session) error {
	task, ok := bandwidthReportManager.tasks[session.SessionKey()]
	if ok && task.IsRunnable() {
		task.Stop(ctx)
	}
	return nil
}

func (bandwidthReportManager *bandwidthReportManager) initialization(ctx context.Context) error {
	var collectors []*collector.BandwidthCollector
	var storeEngine store.BandwidthEngine
	var err error
	defer func() {
		if err != nil {
			for _, collector := range collectors {
				collector.Stop(ctx)
			}
			if storeEngine != nil {
				storeEngine.Stop(ctx)
			}
		}
	}()
	bpfFilter := bandwidthReportManager.reportConfig.BpfFilter
	if len(bpfFilter) == 0 {
		bpfFilter = defaultBpfFilter
	}
	collectors, err = buildBandwidthCollector(bpfFilter)
	if err != nil {
		return err
	}
	engineBufLen := 0
	if bandwidthReportManager.reportConfig != nil {
		engineBufLen = int(bandwidthReportManager.reportConfig.EngineBufLen)
	}

	storeEngine = store.NewBandwidthEngine(collectors, engineBufLen)
	for _, collector := range collectors {
		err = collector.Start(ctx)
		if err != nil {
			return err
		}
	}
	err = storeEngine.Start(ctx)
	if err != nil {
		return err
	}
	bandwidthReportManager.collectors = collectors
	bandwidthReportManager.engine = storeEngine
	return nil
}
func buildBandwidthCollector(bpfFilter string) ([]*collector.BandwidthCollector, error) {
	interfaces, err := nnet.GetValidInterfaces()
	if err != nil {
		return nil, err
	}
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Errorf("pcap findAllDevs err :%v", err)
		return nil, err
	}
	var collectors []*collector.BandwidthCollector
	for _, d := range devices {
		inter := InterfaceSlice(interfaces).FindInterface(d.Addresses)
		if inter != nil {
			collectors = append(collectors,
				collector.NewBandwidthCollector(d.Name, bpfFilter, inter.HardwareAddr.String()))
		}

	}
	return collectors, nil
}

type InterfaceSlice []net.Interface

func (ifs InterfaceSlice) FindInterface(pIfs []pcap.InterfaceAddress) *net.Interface {
	for _, fs := range ifs {
		addrs, err := fs.Addrs()
		if err != nil {
			continue
		}
		var ips []string
		for _, addr := range addrs {
			if a, ok := addr.(*net.IPNet); ok {
				ips = append(ips, a.IP.String())
			}
		}
		for _, pIf := range pIfs {
			if !containStr(ips, pIf.IP.String()) {
				break
			}
			return &fs
		}
	}
	return nil
}
func containStr(set []string, target string) bool {
	for _, str := range set {
		if str == target {
			return true
		}
	}
	return false
}
