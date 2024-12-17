package logfile

import (
	"errors"
	"fmt"
)

type LogSizeExceedErr struct {
	capacity int32
}

func NewLogSizeExceedErr(capacity int32) *LogSizeExceedErr {
	return &LogSizeExceedErr{capacity: capacity}
}

func (e *LogSizeExceedErr) Error() string {
	return fmt.Sprintf("total file size exceeded %d", e.capacity)
}

func IsLogSizeExceedErr(err error) bool {
	if err == nil {
		return false
	}
	var e *LogSizeExceedErr
	return errors.As(err, &e)
}
