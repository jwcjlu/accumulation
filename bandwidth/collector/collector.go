package collector

import (
	"accumulation/bandwidth/model"
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type BandwidthCollector struct {
	deviceName        string
	macAddress        string
	handle            *pcap.Handle
	mutex             *sync.Mutex
	isRunning         atomic.Bool
	bpfFilter         string
	stats             map[string]*model.Bandwidth
	lastCollectorTime int64
}

func NewBandwidthCollector(deviceName, bpfFilter, macAddress string) *BandwidthCollector {
	return &BandwidthCollector{
		deviceName: deviceName,
		macAddress: macAddress,
		bpfFilter:  bpfFilter,
		stats:      map[string]*model.Bandwidth{},
		mutex:      &sync.Mutex{},
	}
}

// Stop pcap capture
func (tc *BandwidthCollector) Stop(ctx context.Context) error {
	defer func() {
		log.Infof("DeviceName %s ,MacAddress %s :Stop success", tc.deviceName, tc.macAddress)
	}()
	tc.isRunning.Swap(false)
	if tc.handle != nil {
		tc.handle.Close()
		tc.handle = nil
	}
	return nil
}

func (tc *BandwidthCollector) Start(ctx context.Context) error {
	tc.mutex.Lock()
	var err error
	defer func() {
		tc.mutex.Unlock()
		log.Infof("DeviceName %s ,MacAddress %s :Start success", tc.deviceName, tc.macAddress)
	}()

	if tc.handle, err = pcap.OpenLive(tc.deviceName, 65535, true, time.Second); err != nil {
		return err
	}
	if len(tc.bpfFilter) > 0 {
		err = tc.handle.SetBPFFilter(tc.bpfFilter)
		if err != nil {
			return err
		}
	}
	tc.isRunning.Swap(true)
	tc.loopReadPacket()
	return nil
}

func (tc *BandwidthCollector) loopReadPacket() {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				log.Errorf("NewPacketSource panic|err=%v|stack=%v", e, string(debug.Stack()))
			}
		}()
		// 开始抓包
		for tc.isRunning.Load() && tc.handle != nil {
			packetData, _, err := tc.handle.ZeroCopyReadPacketData()
			if err != nil {
				log.Errorf("ZeroCopyReadPacketData error err:%v", err)
				continue
			}
			// 只获取以太网帧
			packet := gopacket.NewPacket(packetData, layers.LayerTypeEthernet, gopacket.Default)
			ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
			ipPort := parseIpPortInfo(packet)
			if ethernetLayer != nil && ipPort != nil {
				ethernet := ethernetLayer.(*layers.Ethernet)
				// 如果封包的目的MAC是本机则表示是下行的数据包，否则为上行
				if ethernet.DstMAC.String() == tc.macAddress {
					tc.addPacketLen(len(packet.Data()), ipPort.dstIP, ipPort.dstPort, model.Down)
				} else if ethernet.SrcMAC.String() == tc.macAddress {
					tc.addPacketLen(len(packet.Data()), ipPort.srcIP, ipPort.srcPort, model.Up)
				}
			}
		}
	}()
}

func parseIpPortInfo(packet gopacket.Packet) *IpPortInfo {
	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return nil
	}
	srcIp, dstIp := networkLayer.NetworkFlow().Endpoints()
	// 解析传输层数据
	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return nil
	}
	srcPort := transportLayer.TransportFlow().Src().String()
	dstPort := transportLayer.TransportFlow().Dst().String()
	return &IpPortInfo{
		srcIP:   srcIp.String(),
		srcPort: srcPort,
		dstIP:   dstIp.String(),
		dstPort: dstPort,
	}
}

type IpPortInfo struct {
	srcIP   string
	srcPort string
	dstIP   string
	dstPort string
}

func (tc *BandwidthCollector) addPacketLen(pcapDataLen int, ip string, port string, trafficType model.TrafficType) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	bandwidth, ok := tc.stats[ip+":"+port]
	if !ok {
		bandwidth = &model.Bandwidth{
			MacAddress: tc.macAddress,
			Ip:         ip,
			Port:       port,
			StartTime:  tc.lastCollectorTime,
		}
		tc.stats[ip+":"+port] = bandwidth
	}
	bandwidth.AddPacketLen(int32(pcapDataLen), trafficType)
}

func (tc *BandwidthCollector) ExportAndClean() []*model.Bandwidth {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	var result []*model.Bandwidth
	endTime := time.Now().Unix()
	for _, v := range tc.stats {
		v.CollectTime = endTime
		result = append(result, v)
	}
	tc.lastCollectorTime = endTime
	tc.stats = make(map[string]*model.Bandwidth)
	return result
}
