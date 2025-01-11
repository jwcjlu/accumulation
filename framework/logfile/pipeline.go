package logfile

import (
	"accumulation/pkg/log"
	"context"
	"fmt"

	"runtime"
	"strconv"
	"time"
)

type PipelineBiz struct {
}

func NewPipelineBiz() *PipelineBiz {
	return &PipelineBiz{}
}

func (p *PipelineBiz) Pipeline(ctx context.Context) error {

	return fmt.Errorf("no implement")
}

// UploadLog 上传日志
// 先查看该游戏是否需要上传日志,需要上传日志则进行下面几步
// 1.把需要上传的文件都copy到临时目录
// 2.上传过程可以异步，所以开启一个协程来执行
// 3.归档上传的文件
// 4.上传文件
// 5.清理文件
func (p *PipelineBiz) UploadLog(ctx context.Context, logConfig *StopGameLogConfig) error {
	log.Debugf(ctx, "log upload config:%#v", *logConfig)
	if !logConfig.IsUpload() {
		return nil
	}
	newCtx := WithLogMetricContext(ctx, strconv.FormatInt(logConfig.AreaType, 10),
		strconv.FormatInt(logConfig.GID, 10), strconv.FormatInt(logConfig.VMID, 10))
	archiveTaskDesc, err := logConfig.LogConfig.MoveTask().DoMove(ctx, logConfig.FlowID)
	if err != nil {
		log.Errorf(ctx, "do move failure err:%v", err)
		ReportLogMetric(newCtx, DoMoveFailureCode, 0)
		return nil
	}
	go func(ctx context.Context, desc *ArchiveTaskDesc) {
		defer func() {
			if rerr := recover(); rerr != nil {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				buf = buf[:n]
				log.Errorf(ctx, " %+v\n%s\n", rerr, buf)
			}
		}()
		for i := 0; i < 3; i++ {
			_, err = logConfig.BuildPipeline().Invoke(newCtx, desc)
			if err == nil {
				log.Debugf(ctx, "log upload success")
				return
			}
			if IsLogSizeExceedErr(err) {
				log.Warnf(ctx, "log failure err:%v", err)
				return
			}
			time.Sleep(5 * time.Second)
		}
	}(ctx, archiveTaskDesc)
	return nil
}
