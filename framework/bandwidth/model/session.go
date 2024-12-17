package model

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TrafficType int

const (
	All TrafficType = iota
	Up
	Down
)

type Session struct {
	Start        int64       `json:"start"`
	FlowID       string      `json:"flow_id"`
	BizID        int64       `json:"biz_id"`
	GID          int64       `json:"gid"`
	UUID         string      `json:"uuid"`
	VMid         int64       `json:"vmid"`
	AreaType     int32       `json:"area_type"`
	InstanceId   string      `json:"instance_id"` //实例ID
	Idc          string      `json:"idc"`         //机房
	StreamIp     string      `json:"stream_ip"`
	StreamPorts  StreamPorts `json:"stream_port"`
	Extra        string      `json:"extra"`
	EIP          int32       `json:"eip"`
	ImageVersion int         `json:"image_version"`
}

func (session *Session) String() string {
	if session == nil {
		return ""
	}
	data, _ := json.Marshal(session)
	return string(data)
}

type StreamPorts []StreamPort

func (sps StreamPorts) Contains(port string) bool {
	for _, sp := range sps {
		if strconv.Itoa(sp.Port) == port {
			return true
		}
	}
	return false
}
func (session *Session) SessionKey() string {
	return unique(session.InstanceId, session.VMid)
}
func (session *Session) ReportId() string {
	return fmt.Sprintf("%s-%d-%d-%s", session.InstanceId, session.BizID, session.VMid, session.FlowID)
}
func unique(instanceId string, vmID int64) string {
	return fmt.Sprintf("%s-%d", instanceId, vmID)
}
