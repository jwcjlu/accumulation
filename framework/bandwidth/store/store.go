package store

import (
	"accumulation/framework/bandwidth/model"
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	cleanOffsetThreshold = 1024 * 1024
)

type FileStore[T model.Serialization[T]] struct {
	store  *os.File
	path   string
	offset int64
	mutex  *sync.Mutex
}

func NewFileStore[T model.Serialization[T]](path string) *FileStore[T] {
	return &FileStore[T]{
		path:  path,
		mutex: &sync.Mutex{},
	}
}

func (fs *FileStore[T]) Open() (err error) {
	parentDir := filepath.Dir(fs.path)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory:%v", err)
	}

	if fs.store == nil {
		if fs.store, err = os.OpenFile(fs.path, os.O_RDWR|os.O_CREATE, 0755); err != nil {
			return err
		}
	}
	return nil
}
func (fs *FileStore[T]) Close() (err error) {
	return fs.store.Close()
}
func (fs *FileStore[T]) Store(ctx context.Context, reqs []T) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	return fs.doWrite(ctx, reqs)

}
func (fs *FileStore[T]) doWrite(ctx context.Context, reqs []T) error {
	writer := bufio.NewWriter(fs.store)
	for _, req := range reqs {
		data, err := req.Encode()
		if err != nil {
		}
		if err = NewRecord(data).write(writer); err != nil {
			return err
		}
	}
	return writer.Flush()

}
func (fs *FileStore[T]) Load(ctx context.Context, rows int) ([]T, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	fs.store.Seek(fs.offset, io.SeekStart)
	reader := bufio.NewReader(fs.store)
	var reqs []T
	index := 0
	var record *Record
	var err error
	dataLen := 0
	for record, err = read(reader); index < rows && record != nil && err == nil; record, err = read(reader) {
		var r T
		req := r.Instance()
		if err = req.Decode(record.data); err != nil {
			if err = fs.store.Truncate(0); err != nil {
				return nil, err
			}
			if _, err = fs.store.Seek(0, 0); err != nil {
				return nil, err
			}
			fs.offset = 0
			return reqs, nil
		}
		reqs = append(reqs, req)
		dataLen += record.DataLen()
		index++
	}
	fs.offset += int64(dataLen)
	if int64(cleanOffsetThreshold) < fs.offset {
		fs.Truncate(ctx, nil, nil)
	}
	return reqs, nil
}

func (fs *FileStore[T]) Truncate(ctx context.Context, before, after []T) error {
	fileInfo, err := fs.store.Stat()
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()
	keepLength := fileSize - fs.offset
	// 读取需要保留的内容
	data := make([]byte, keepLength)
	_, err = fs.store.ReadAt(data, fs.offset)
	if err != nil {
		return err
	}
	// 截断文件到保留内容的长度
	if err = fs.store.Truncate(0); err != nil {
		return err
	}
	if _, err = fs.store.Seek(0, 0); err != nil {
		return err
	}
	if err = fs.doWrite(ctx, before); err != nil {
		return err
	}
	if _, err = fs.store.Write(data); err != nil {
		return err
	}
	if err = fs.doWrite(ctx, after); err != nil {
		return err
	}
	fs.offset = 0
	return nil
}

type Record struct {
	dLen int16
	data []byte
}

func (r *Record) DataLen() int {
	return len(r.data) + 2
}
func NewRecord(data []byte) *Record {
	return &Record{
		dLen: int16(len(data)),
		data: data,
	}
}

func (r *Record) write(w *bufio.Writer) error {
	dLen := make([]byte, 2)
	dLen[0] = byte(r.dLen)
	if _, err := w.Write(dLen); err != nil {
		return err
	}
	if _, err := w.Write(r.data); err != nil {
		return err
	}
	return nil
}
func read(r *bufio.Reader) (*Record, error) {
	head, err := readFixedLength(r, 2)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	dLen := int16(binary.LittleEndian.Uint16(head))

	data, err := readFixedLength(r, int(dLen))
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Record{
		data: data,
		dLen: dLen,
	}, nil

}

// 每次buffer都是固定长度，这里就有可能导致一个结构体对象不完全在当次的buff里面，需要二次buff进行拼接
// 这里有个前提是一个结构体的大小小于4096，如果对象大于这个值需要修改默认大小或多次读入
func readFixedLength(r *bufio.Reader, n int) ([]byte, error) {
	data := make([]byte, n)
	remaining := 0
	if remaining = r.Buffered(); remaining < n {
		data = make([]byte, remaining)
	}
	_, err := r.Read(data)
	if err != nil {
		return nil, err
	}
	if remaining < n {
		remainData := make([]byte, n-remaining)
		_, err = r.Read(remainData)
		if err != nil {
			return nil, err
		}
		data = append(data, remainData...)
	}
	return data, nil
}
