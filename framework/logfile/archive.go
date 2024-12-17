package logfile

import (
	"accumulation/pkg/log"
	"archive/zip"
	"context"
	"fmt"

	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ArchiveTask struct {
	desc        *ArchiveTaskDesc
	archiveFile string
}
type ArchiveTaskDesc struct {
	Files       []*FileDesc ` json:"files"`
	ArchiveType ArchiveType `json:"archive_type"`
	ArchiveFile string      ` json:"archive_file"`
}

func NewArchiveTask() Handler {
	return &ArchiveTask{}
}

// Filter 文件过滤器
type Filter func(fi os.FileInfo) bool

type Archive interface {
	Archive(ctx context.Context, infos []*FileDesc, archiveFile string)
}

func (task *ArchiveTask) Rollback() {
}
func (task *ArchiveTask) Type() TaskType {
	return TaskType_ARCHIVE
}
func (task *ArchiveTask) Do(ctx context.Context, input interface{}) (interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("archive file is required")
	}
	desc, ok := input.(*ArchiveTaskDesc)
	if !ok {
		return nil, fmt.Errorf("archive file need string,however input[%v]", input)
	}
	task.desc = desc
	archiveFile := desc.ArchiveFile
	if len(archiveFile) == 0 {
		archiveFile = task.desc.ArchiveFile
	}
	if len(archiveFile) == 0 {
		return nil, fmt.Errorf("archiveFile is empty")
	}
	task.archiveFile = archiveFile
	switch task.desc.ArchiveType {
	case ArchiveType_ZIP:
		return zipArchive(ctx, task.desc.Files, archiveFile)
	}
	return nil, fmt.Errorf("not found archive type")
}

// zipArchive 归档
func zipArchive(ctx context.Context, infos []*FileDesc, archiveFile string) (fi *FileDesc, err error) {
	if f, ok := fileExist(archiveFile); ok {
		return &FileDesc{
			FileType: FileType_FILE,
			ModTime:  uint32(f.ModTime().Unix()),
			Size:     int32(f.Size()),
			Name:     archiveFile,
		}, nil
	}
	fw, err := createIfNeeded(archiveFile)
	if err != nil {
		return nil, err
	}
	zw := zip.NewWriter(fw)
	defer func() {
		// 检测一下是否成功关闭
		if err = zw.Close(); err != nil {
			log.Errorf(ctx, "close file[v%] err[v%]", archiveFile, err)
		}
		fw.Close()
	}()
	for _, fileInfo := range infos {
		exist, _ := pathExists(fileInfo.Dir)
		if !exist { //不存在跳过
			continue
		}
		err = doArchive(zw, fileInfo.Dir, func(fi os.FileInfo) bool {
			matched, err := filepath.Match(fileInfo.Wildcard, fi.Name())
			if err != nil {
				return false
			}
			if matched && uint32(time.Now().Unix()-fi.ModTime().Unix()) < fileInfo.ModTime {
				return true
			}
			return false
		})
		if err != nil {
			return nil, err
		}
	}
	f, err := fw.Stat()
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		rule := &FileFilterRule{
			Dir:   info.Dir,
			Regex: info.Wildcard,
		}
		rule.remove(ctx, func(errCtx context.Context, fileName string, err error) bool {
			if err != nil {
				log.Errorf(ctx, "fileName %v remove failure but ignore,Err%v", fileName, err)
			}
			return true
		})
	}
	return &FileDesc{
		FileType: FileType_FILE,
		ModTime:  uint32(f.ModTime().Unix()),
		Size:     int32(f.Size()),
		Name:     archiveFile,
	}, nil
}

func doArchive(zw *zip.Writer, src string, fileFilter Filter) (err error) {

	// 下面来将文件写入 zw ，因为有可能会有很多个目录及文件，所以递归处理
	return filepath.Walk(src, func(path string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}
		// 通过文件信息，创建 zip 的文件信息
		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return
		}
		// 替换文件信息中的文件名
		fh.Name = strings.TrimPrefix(filepath.ToSlash(path), src)
		if fi.IsDir() {
			fh.Name += "/"
			return nil
		} else {
			if !fileFilter(fi) { //过滤掉不符合的文件类型
				return nil
			}
		}
		// 写入文件信息，并返回一个 Write 结构
		w, err := zw.CreateHeader(fh)
		if err != nil {
			return
		}
		// 检测，如果不是标准文件就只写入头信息，不写入文件数据到 w
		// 如目录，也没有数据需要写
		if !fh.Mode().IsRegular() {
			return nil
		}
		// 打开要压缩的文件
		fr, err := os.Open(path)
		defer func() {
			fr.Close()
		}()
		if err != nil {
			return
		}

		// 将打开的文件 Copy 到 w
		_, err = io.Copy(w, fr)
		if err != nil {
			return fmt.Errorf("src: %s, dst: %s copy err %v", src, path, err)
		}

		return nil
	})
}

func createIfNeeded(path string) (*os.File, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(path), os.ModePerm)
		} else {
			return nil, err
		}

	}
	return os.Create(path)
}
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
