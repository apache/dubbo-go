package filesystem

import (
	perrors "github.com/pkg/errors"
	"os"
)

type File struct {
	Path         string   //路径
	fileWriter   *os.File //写入文件
}

// 创建文件锁，配合 defer f.Release() 来使用
func (file *File)Create() (f *Filelock, e error) {
	if file == nil || file.Path == "" {
		return nil, perrors.Errorf("cannot create flock on empty path")
	}
	lock, e := os.Create(file.Path)
	if e != nil {
		return
	}
	return &Filelock{
		LockFile: file,
		lock:     lock,
	}, nil
}
