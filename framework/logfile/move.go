package logfile

import (
	"accumulation/pkg/log"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	Dir = "logtmp"
)

type MoveTask struct {
	fileFilterRules       []FileFilterRule
	temp                  string
	uploadTimeRecentLimit int32
	isDeleteSourceFile    bool
}

func NewMoveTask(fileFilterRules []FileFilterRule, uploadTimeRecentLimit int32, isDeleteSourceFile bool) *MoveTask {
	if uploadTimeRecentLimit == 0 {
		uploadTimeRecentLimit = 30 * 24 * 3600 * 24
	}
	return &MoveTask{
		fileFilterRules:       fileFilterRules,
		uploadTimeRecentLimit: uploadTimeRecentLimit,
		isDeleteSourceFile:    isDeleteSourceFile,
	}
}

// DoMove 先把上传的文件全部拷贝到临时目录
func (task *MoveTask) DoMove(ctx context.Context, tmpDir string) (*ArchiveTaskDesc, error) {
	go CleanExpiredData(ctx)
	pwd, _ := os.Getwd()
	dstPath := filepath.ToSlash(filepath.Join(pwd, Dir, tmpDir))
	archiveFile := filepath.Join(pwd, Dir, fmt.Sprintf("%s.zip", tmpDir))
	archiveTaskDesc := &ArchiveTaskDesc{
		Files:       []*FileDesc{{Dir: dstPath, Wildcard: "*", ModTime: 24 * 3600}},
		ArchiveType: ArchiveType_ZIP,
		ArchiveFile: archiveFile,
	}
	//如果已经打包好了，直接跳过
	if _, ok := fileExist(archiveFile); ok {
		return archiveTaskDesc, nil
	}
	err := MkdirIfNeeded(dstPath)
	if err != nil {
		return nil, fmt.Errorf("MkdirIfNeeded err:%v", err)
	}
	for _, fileFilterRule := range task.fileFilterRules {
		srcPath := fileFilterRule.GetDir()
		if err = task.copyDir(ctx, srcPath, dstPath, fileFilterRule); err != nil {
			return nil, err
		}
	}
	mkdirEmptyFileIfNeeded(ctx, dstPath)
	if task.isDeleteSourceFile {
		for _, file := range task.fileFilterRules {
			if err = file.remove(ctx, func(errCtx context.Context, fileName string, err error) bool {
				if osErr, ok := err.(*os.PathError); ok {
					log.Warnf(ctx, "remove file[%s] failure :err:%v", fileName, osErr)
					return true
				}
				return false

			}); err != nil {
				return nil, fmt.Errorf("remove rsource err:%v", err)
			}
		}
	}
	return archiveTaskDesc, nil
}

func mkdirEmptyFileIfNeeded(ctx context.Context, path string) {
	dir, err := os.Open(path)
	if err != nil {
		log.Errorf(ctx, "open directory [%s] failure err:%v", path, err)
		return
	}
	defer dir.Close()
	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		log.Errorf(ctx, "readdir directory [%s] failure err:%v", path, err)
		return
	}
	if len(fileInfos) == 0 {
		os.Create(fmt.Sprintf(filepath.Join(path, "empty")))
	}
}

func (task *MoveTask) copyDir(ctx context.Context, srcPath, dstPath string, fileFilterRule FileFilterRule) error {
	_, err := os.Stat(srcPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf(ctx, "file [%s] is not exist", srcPath)
			return nil
		}
		return fmt.Errorf("stat srcPath %s err:%v", srcPath, err)
	}
	fileFilter := func(regex string, fi os.FileInfo) bool {
		matched, err := filepath.Match(regex, fi.Name())
		if err != nil {
			return false
		}
		if matched && int32(time.Now().Unix()-fi.ModTime().Unix()) < task.uploadTimeRecentLimit {
			return true
		}
		return false
	}
	return filepath.Walk(srcPath, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		} else {
			if !fileFilter(fileFilterRule.Regex, info) { //过滤掉不符合的文件类型
				return nil
			}
		}
		trimPath := srcPath
		if !isDir(trimPath) {
			trimPath = filepath.ToSlash(filepath.Dir(trimPath))
		}
		if os.IsPathSeparator(path[len(trimPath)-1]) {
			trimPath = filepath.Dir(trimPath)
		}
		//返回上一层目录
		if tmp := filepath.Dir(trimPath); len(tmp) > 2 {
			trimPath = filepath.ToSlash(tmp)
		}
		path = filepath.ToSlash(path)
		dstFilePath := filepath.Join(dstPath, strings.TrimPrefix(path, trimPath))
		err = MkdirIfNeeded(filepath.Dir(dstFilePath))
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(dstFilePath, os.O_RDWR|os.O_CREATE, 0644)
		defer dst.Close()
		if err != nil {
			return fmt.Errorf("open dstPath %s:err:%v", dstFilePath, err)
		}
		src, err := os.Open(path)
		defer src.Close()
		if err != nil {
			return fmt.Errorf("open srcPath %s:err:%v", path, err)
			return err
		}
		_, err = io.Copy(dst, src)
		if err == nil {
			log.Debugf(ctx, "[%s]file copy to [%s] success ", path, dstFilePath)
		}

		return err
	})
}

func MkdirIfNeeded(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// 目录不存在，创建目录
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// 检查文件是否存在
func fileExist(file string) (os.FileInfo, bool) {
	f, err := os.Stat(file)
	if err != nil {
		return nil, false
	}
	return f, true
}

func isDir(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	// 检查是否是文件
	if !fileInfo.Mode().IsRegular() {
		return true
	}
	return false
}

var expiredTime int64 = 8 * 60 * 60
var cleanExpiredDataLock = sync.Mutex{}

func CleanExpiredData(ctx context.Context) {
	if !cleanExpiredDataLock.TryLock() {
		return
	}
	defer cleanExpiredDataLock.Unlock()
	pwd, _ := os.Getwd()
	dstPath := filepath.ToSlash(filepath.Join(pwd, Dir))
	dir, err := os.Open(dstPath)
	if err != nil {
		log.Errorf(ctx, "open directory [%s] failure err:%v", dstPath, err)
		return
	}
	defer dir.Close()
	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		log.Errorf(ctx, "readdir directory [%s] failure err:%v", dstPath, err)
		return
	}
	for _, fileInfo := range fileInfos {
		if time.Now().Unix()-fileInfo.ModTime().Unix() < expiredTime {
			continue
		}
		if fileInfo.IsDir() {
			os.RemoveAll(filepath.Join(dstPath, fileInfo.Name()))
		} else {

			os.Remove(filepath.Join(dstPath, fileInfo.Name()))
		}
	}
}
