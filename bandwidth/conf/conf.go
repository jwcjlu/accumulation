package conf

import (
	"google.golang.org/protobuf/types/known/durationpb"
)

type Acl_ReportConfig struct {
	Host                  string               `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	BackendReportInterval int32                `protobuf:"varint,2,opt,name=backend_report_interval,json=backendReportInterval,proto3" json:"backend_report_interval,omitempty"` //上报周期
	BpfFilter             string               `protobuf:"bytes,3,opt,name=bpf_filter,json=bpfFilter,proto3" json:"bpf_filter,omitempty"`
	SessionTimeout        *durationpb.Duration `protobuf:"bytes,4,opt,name=session_timeout,json=sessionTimeout,proto3" json:"session_timeout,omitempty"`
	ReportJobBufLen       int32                `protobuf:"varint,5,opt,name=report_job_buf_len,json=reportJobBufLen,proto3" json:"report_job_buf_len,omitempty"` //report 的buf长度
	EngineBufLen          int32                `protobuf:"varint,6,opt,name=engine_buf_len,json=engineBufLen,proto3" json:"engine_buf_len,omitempty"`            //engine的buf长度
}
type Acl struct {
	ReportConfig *Acl_ReportConfig `protobuf:"bytes,6,opt,name=reportConfig,proto3" json:"reportConfig,omitempty"`
}
type Data struct {
	Debug bool `protobuf:"varint,1,opt,name=debug,proto3" json:"debug,omitempty"`
	Acl   *Acl `protobuf:"bytes,3,opt,name=acl,proto3" json:"acl,omitempty"` //acl相关外部服务信息配置
}
