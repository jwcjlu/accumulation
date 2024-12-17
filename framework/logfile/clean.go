package logfile

import (
	"context"
	"fmt"
)

type CleanTask struct {
}

func NewCleanTask() Handler {
	return &CleanTask{}
}

func (task *CleanTask) Type() TaskType {
	return TaskType_CLEAN
}
func (task *CleanTask) Do(ctx context.Context, input interface{}) (interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("archive file is required")
	}
	files, ok := input.([]*FileFilterRule)
	if !ok {
		return nil, fmt.Errorf("archive file need string,however input[%v]", input)
	}
	for _, fileInfo := range files {
		if err := fileInfo.remove(ctx, func(errCtx context.Context, fileName string, err error) bool {
			return false
		}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (task *CleanTask) Rollback() {
}
