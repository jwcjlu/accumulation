package model

import (
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tidwall/gjson"

	"sort"
)

var Sep = "|"

type Bandwidth struct {
	MacAddress  string
	Ip          string
	Port        string
	UpLen       int32
	DownLen     int32
	StartTime   int64
	CollectTime int64
}

func (b *Bandwidth) Print() {
	log.Debugf("mac:%s,ip:%s,up:%d,down:%d,collectTime:%d", b.MacAddress,
		b.Ip, b.UpLen, b.DownLen, b.CollectTime)
}
func (b *Bandwidth) AddPacketLen(pcapDataLen int32, trafficType TrafficType) {
	switch trafficType {
	case Up:
		b.UpLen += pcapDataLen
		return
	case Down:
		b.DownLen += pcapDataLen
	}
}

type Bandwidths []*Bandwidth

func NewBandwidths(bws []*Bandwidth) Bandwidths {
	return bws
}
func (bws Bandwidths) Len() int {
	return len(bws)
}

func (bws Bandwidths) Less(i, j int) bool {
	return bws[i].CollectTime < bws[j].CollectTime
}

func (bws Bandwidths) Swap(i, j int) {
	bws[i], bws[j] = bws[j], bws[i]
}
func (bws Bandwidths) Sort() {
	sort.Sort(bws)
}

type Filters func(bandwidth *Bandwidth) bool

func (bws Bandwidths) Filter(filters Filters) []*Bandwidth {
	var data []*Bandwidth
	for _, d := range bws {
		if filters(d) {
			data = append(data, d)
		}
	}
	return data
}

// Search 二分搜索算法
func (bws Bandwidths) Search(startTime, endTime int64) []*Bandwidth {
	startIndex := sort.Search(len(bws), func(i int) bool {
		return bws[i].CollectTime >= startTime
	})
	endIndex := len(bws)
	if endTime > 0 {
		endIndex = sort.Search(len(bws), func(i int) bool {
			return bws[i].CollectTime >= endTime
		})
	}
	return bws[startIndex:endIndex]
}

func (bws Bandwidths) Group() (upstream, downstream int32) {
	for _, traffic := range bws {
		upstream += traffic.UpLen
		downstream += traffic.DownLen
	}
	return
}

type GameStarted struct {
	Start        int64        `json:"start"`
	FlowID       string       `json:"flow_id"`
	BizID        int64        `json:"biz"`
	GID          int64        `json:"gid"`
	UUID         int64        `json:"uuid"`
	VMid         int64        `json:"vmid"`
	AreaType     int32        `json:"area_type"`
	InstanceId   string       `json:"instance_id"` //实例ID
	Idc          string       `json:"idc"`         //机房
	StreamIp     string       `json:"stream_ip"`
	StreamPorts  []StreamPort `json:"stream_ports"`
	Extra        string       `json:"extra"`
	ImageVersion int          `json:"image_version"`
	RuntimeInfo  string       `json:"runtime_info"`
}

func (gameStarted *GameStarted) EIP() int32 {
	if len(gameStarted.RuntimeInfo) == 0 {
		return 0
	}
	eip := gjson.Get(gameStarted.RuntimeInfo, "run_game_tag")
	if len(eip.Raw) == 0 {
		return 0
	}
	return int32(eip.Int())
}

type DevRuntimeInfo struct {
	EIP int32 `json:"run_game_tag,omitempty"` //eip
}

type StartGameResp struct {
	StreamIp    string       `json:"stream_ip"`
	StreamPorts []StreamPort `json:"stream_ports"`
}

type StreamPort struct {
	Name         string `json:"name"`
	ProtocolType string `json:"protocol_type"`
	Port         int    `json:"stream_port"`
}

func (gameStarted *GameStarted) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, gameStarted)
	if err != nil {
		return fmt.Errorf("err:%v,body:%s", err, string(data))
	}
	log.Debugf("gameStarted body :%v", string(data))
	return nil
}

type GameStop struct {
	Start      int64  `json:"start"`
	FlowID     string `json:"flow_id"`
	BizID      int64  `json:"biz_id"`
	GID        int64  `json:"gid"`
	UUID       string `json:"uuid"`
	VMid       int64  `json:"vmid"`
	AreaType   int32  `json:"area_type"`
	InstanceId string `json:"instance_id"` //实例ID
}

func (gameStop *GameStop) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, gameStop)
	if err != nil {
		return fmt.Errorf("err:%v,body:%s", err, string(data))
	}
	return nil
}

type Serialization[R any] interface {
	Encode() ([]byte, error)
	Decode(data []byte) error
	Instance() R
}
type ReportFlowBizRequest struct {
}

func (r *ReportFlowBizRequest) Encode() ([]byte, error) {
	return nil, nil /*proto.Marshal(&r.ReportFlowBizRequest)*/
}

func (r *ReportFlowBizRequest) Decode(data []byte) error {
	/*	if err := proto.Unmarshal(data, &r.ReportFlowBizRequest); err != nil {
		return err
	}*/
	return nil
}
func (r *ReportFlowBizRequest) Instance() *ReportFlowBizRequest {
	return &ReportFlowBizRequest{}
}
