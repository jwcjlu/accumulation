package logfile

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

type FileType int32

const (
	FileType_DIR       FileType = 1 // 快手-系统目录
	FileType_FILE      FileType = 2 // 快手-游戏目录
	FILE_TYPE_EXE      FileType = 3 // 字节-相对exe路径
	FILE_TYPE_ABSOLUTE FileType = 4 // 字节-绝对路径
	FILE_TYPE_USR      FileType = 5 // 字节-相对usr文件路径
)

type ArchiveType int32

const (
	ArchiveType_ZIP ArchiveType = 1
	ArchiveType_TAR ArchiveType = 2
)

type TaskType int32

const (
	TaskType_UPLOAD   TaskType = 1
	TaskType_DOWNLOAD TaskType = 2
	TaskType_ARCHIVE  TaskType = 3
	TaskType_CLEAN    TaskType = 4
)

type ServerType int32

const (
	ServerType_HTTP ServerType = 1
	ServerType_FTP  ServerType = 2
)

type AuthType int32

const (
	AuthType_USER_PWD AuthType = 1
)

type Manufacturer int32

const (
	Manufacturer_KWAI      Manufacturer = 1
	Manufacturer_ByteDance Manufacturer = 2
)

type StopGameLogConfig struct {
	LogConfig LogConfig `json:"log_config"`
	FlowID    string    `json:"flow_id"`
	AreaType  int64     `json:"area_type"`
	GID       int64     `json:"gid"`
	VMID      int64     `json:"vmid"`
}

func (config *StopGameLogConfig) IsUpload() bool {
	return config != nil && config.LogConfig.Status == 2
}

type LogConfig struct {
	RemoteProducer        Manufacturer      `json:"remote_producer"`          // 上传产商，1-快手
	RemotePath            string            `json:"remote_path"`              // 远程上传路径
	RemoteUrl             string            `json:"remote_url"`               // 远程上传域名
	UploadTimeCostLimit   int32             `json:"upload_time_cost_limit"`   // 上传整体耗时限制，单位秒
	UploadFlowLimit       int32             `json:"upload_flow_limit"`        // 上传流量限制，单位 KB/s, 100代表100KB/s
	UploadSizeLimit       int32             `json:"upload_size_limit"`        // 上传文件总限制大小，单位KB
	UploadTimeRecentLimit int32             `json:"upload_time_recent_limit"` // 上传文件的时效要求，单位秒。60代表最近一分钟的文件才需要上传
	AuthType              AuthType          `json:"auth_type"`                // 鉴权方法。1-secret密钥.....
	AuthInfo              string            `json:"auth_info"`                // 鉴权参数，json字段
	UploadMethod          ServerType        `json:"upload_method"`            // 上传方法。1-http，2-ftp.....
	Status                int               `json:"status"`                   //2:表示开启，
	UploadRetryLimit      int               `json:"upload_retry_limit"`       //上传发起连接失败重试次数
	FileFilterRules       []FileFilterRule  `json:"file_filter_rules"`        // 文件过滤规则，描述具体上传哪些文件
	Extra                 map[string]string `json:"extra"`                    // 扩展字段
	IsDeleteSourceFile    bool              `json:"is_delete_source_file"`
}

func (config *LogConfig) MoveTask() *MoveTask {
	return NewMoveTask(config.FileFilterRules, config.UploadTimeRecentLimit, config.IsDeleteSourceFile)
}

type FileFilterRule struct {
	Dir      string   `json:"dir"` // 目录
	Regex    string   `json:"regex"`
	FileType FileType `json:"file_type"`
}

func (rule *FileFilterRule) remove(ctx context.Context, ignoreErr func(errCtx context.Context, fileName string, err error) bool) error {
	dir := rule.GetDir()
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fileInfo.IsDir() {
		err = rmDir(ctx, dir, rule.Regex, ignoreErr)
	} else {
		err = rmFile(dir)
	}
	return err
}

func (rule *FileFilterRule) GetDir() (dir string) {
	defer func() {
		dir = filePathNormalization(dir)
	}()
	if !IsWindows() {
		return rule.Dir
	}
	switch rule.FileType {
	case FILE_TYPE_USR:
		u, _ := user.Current()
		return filepath.Join(u.HomeDir, rule.Dir)
	case FILE_TYPE_EXE:
		return addVolumeIfNeed(rule.Dir)
	case FILE_TYPE_ABSOLUTE:
		return addVolumeIfNeed(rule.Dir)
	}
	return rule.Dir
}
func filePathNormalization(dir string) string {
	tmp := strings.ReplaceAll(filepath.ToSlash(dir), "//", "/")
	for strings.Contains(tmp, "//") {
		tmp = strings.ReplaceAll(filepath.ToSlash(tmp), "//", "/")
	}
	return tmp
}
func (config *StopGameLogConfig) BuildPipeline() *Pipeline {
	pipeline := NewPipeline()

	pipeline.AddHandler(NewArchiveTask())
	pipeline.AddHandler(NewUploadTask(&UploadTaskDesc{
		UploadServer: &ServerDesc{
			Addr:               config.LogConfig.RemoteUrl,
			Path:               config.LogConfig.RemotePath,
			ServerType:         config.LogConfig.UploadMethod,
			AuthenticationInfo: config.LogConfig.AuthInfo,
			Manufacturer:       config.LogConfig.RemoteProducer,
		},
		Limit:    config.LogConfig.UploadFlowLimit,
		Capacity: config.LogConfig.UploadSizeLimit,
		Timeout:  config.LogConfig.UploadTimeCostLimit,
		Attrs:    config.LogConfig.Extra,
	}))
	pipeline.AddHandler(NewCleanTask())
	return pipeline
}

type Task struct {
	TaskType TaskType ` json:"task_type,omitempty"`
	Data     []byte   ` json:"data,omitempty"`
}

func (t Task) GetTaskType() TaskType {
	return t.TaskType
}

func (t Task) GetData() []byte {
	return t.Data
}

type FileDesc struct {
	Dir      string   ` json:"dir,omitempty"`
	Wildcard string   ` json:"wildcard,omitempty"`
	FileType FileType ` json:"file_type,omitempty"`
	ModTime  uint32   ` json:"mod_time,omitempty"`
	Size     int32    ` json:"size,omitempty"`
	Name     string   ` json:"name,omitempty"`
}

type ServerDesc struct {
	Addr               string       ` json:"addr,omitempty"`
	Path               string       ` json:"path,omitempty"`
	ServerType         ServerType   ` json:"server_type,omitempty"`
	AuthenticationInfo string       ` json:"authentication_info,omitempty"`
	Manufacturer       Manufacturer ` json:"manufacturer,omitempty"`
}

func (config *StopGameLogConfig) Unmarshal(data []byte) error {
	err := json.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("err:%v,body:%s", err, string(data))
	}
	return nil
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func addVolumeIfNeed(filePath string) string {
	volume := filepath.VolumeName(filePath)
	if len(volume) > 0 {
		return filePath
	}
	drives := []string{"E:", "G:", "I:"}
	for _, drive := range drives {
		_, err := os.Stat(drive)
		if err == nil {
			return filepath.Join(drive, filePath)
		}
	}
	return ""
}
func rmDir(ctx context.Context, dir, regex string, ignoreErr func(errCtx context.Context, fileName string, err error) bool) error {
	if regex == "*" {
		err := os.RemoveAll(dir)
		if err == nil {
			return nil
		}
		if !ignoreErr(ctx, dir, err) {
			return nil
		}
	}
	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		matched, err := filepath.Match(regex, info.Name())
		if err != nil && !ignoreErr(ctx, path, err) {
			return err
		}
		if matched {
			err = os.Remove(path)
			if !ignoreErr(ctx, path, err) {
				return err
			}
		}
		return nil
	})

}

func rmFile(file string) error {
	return os.Remove(file)
}
