package logfile

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/juju/ratelimit"
)

type UploadTask struct {
	desc *UploadTaskDesc
}
type UploadTaskDesc struct {
	UploadServer *ServerDesc       `json:"upload_server,omitempty"`
	Limit        int32             `json:"limit,omitempty"`
	Capacity     int32             `json:"capacity,omitempty"`
	Timeout      int32             `json:"timeout,omitempty"`
	Attrs        map[string]string `json:"attrs"`
}

func NewUploadTask(desc *UploadTaskDesc) Handler {
	if desc.Capacity == 0 {
		desc.Capacity = 50 * 1024 * 1024
	}
	return &UploadTask{
		desc: desc,
	}
}
func (task *UploadTask) Type() TaskType {
	return TaskType_UPLOAD
}
func (task *UploadTask) Do(ctx context.Context, input interface{}) (interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("input is required")
	}
	file, ok := input.(*FileDesc)
	if !ok {
		return nil, fmt.Errorf("uploading components requires  FileDesc,however input[%v]", input)
	}
	uploadClient := createUploadClient(task.desc.UploadServer)
	if uploadClient == nil {
		return nil, fmt.Errorf("not found upload client uploadServer[%v]", *task.desc.UploadServer)
	}
	if file.Size > task.desc.Capacity {
		ReportLogMetric(ctx, LogSizeExceed, float64(file.Size))
		return nil, NewLogSizeExceedErr(task.desc.Capacity)
	}
	var opts []UpdateOption
	if task.desc.Limit > 0 {
		opts = append(opts, WithRateLimit(func(r io.Reader) io.Reader {
			bucket := ratelimit.NewBucketWithQuantum(1*time.Second, int64(task.desc.Limit), int64(task.desc.Limit))
			return ratelimit.Reader(r, bucket)
		}))
	}
	if task.desc.Timeout > 0 {
		opts = append(opts, WithUploadTimeout(time.Duration(task.desc.Timeout)*time.Second))
	}
	err := uploadClient.UploadFile(ctx, file.Name, task.desc.Attrs, opts...)
	if err != nil {
		ReportLogMetric(ctx, UploadFailureCode, float64(file.Size))
	} else {
		ReportLogMetric(ctx, Success, float64(file.Size))
	}
	return []*FileFilterRule{{
		Dir:      file.Name,
		FileType: FileType_FILE,
		Regex:    "*",
	}}, err
}
func (task *UploadTask) Rollback() {

}

type UploadClient interface {
	UploadFile(ctx context.Context, file string, extra map[string]string, opts ...UpdateOption) error
}

func createUploadClient(desc *ServerDesc) UploadClient {
	return nil
}

type UploadOptions struct {
	timeout   time.Duration
	rateLimit func(r io.Reader) io.Reader
}

type UpdateOption func(o *UploadOptions)

func WithUploadTimeout(timeout time.Duration) UpdateOption {
	return func(o *UploadOptions) { o.timeout = timeout }
}

func WithRateLimit(rateLimit func(r io.Reader) io.Reader) UpdateOption {
	return func(o *UploadOptions) { o.rateLimit = rateLimit }
}
