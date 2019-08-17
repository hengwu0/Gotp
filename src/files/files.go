package files

import (
	"crypto/md5"
	"errors"
	"gopkg.in/cheggaaa/pb.v1"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	FILE = iota
	DIR
	NOEXIST
	LnFILE
	LnDIR
	UNKNOWN
)

var FTYPE = [...]string{"FILE", "DIR", "NOEXIST", "LnFILE", "LnDIR", "UNKNOWN"}

func IsBusy(err error) bool {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err == syscall.ETXTBSY
	case *os.LinkError:
		return false
	case *os.SyscallError:
		return false
	}
	return false
}

func GetCurrentDirectory() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return strings.Replace(dir, "\\", "/", -1), nil
}

func RecvPathCheck(path string) (string, error) {
	if ftype, _ := FileType(path); ftype == DIR {
		return filepath.Abs(path)
	} else {
		return "", errors.New("Can't use path: " + path + ", PathType: " + FTYPE[ftype])
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func DirCreate(pathname string, mode uint32) bool {
	flag, err := PathExists(pathname)
	if err != nil {
		return false
	}
	if flag == false {
		os.MkdirAll(pathname, os.FileMode(mode))
	}
	return true
}

func FileType(filename string) (int, uint32) {
	flag, err := PathExists(filename)
	if err != nil {
		return UNKNOWN, 0
	}
	if flag == false {
		return NOEXIST, 0
	} else {
		f, _ := os.Lstat(filename)
		//fmt.Println("f.sys= ", f.Sys())
		switch {
		case f.IsDir():
			return DIR, uint32(f.Mode())
		default:
			return FILE, uint32(f.Mode())
		}
	}
}

func FileSize(filename string) int64 {
	f, err := os.Lstat(filename)
	if err != nil || f.IsDir() {
		return 0
	}
	return f.Size()
}

func Filename(filename string) string {
	return path.Base(filename)
}

func Filedir(filename string) string {
	return path.Dir(filename)
}

func GetMd5sum(filename string) []byte {
	f, _ := os.Open(filename)

	defer f.Close()
	md5hash := md5.New()
	io.Copy(md5hash, f)

	return md5hash.Sum(nil)
}

//从io.Copy拷贝并重构
func Copy(dst io.Writer, src io.Reader, pbar *pb.ProgressBar) (md5sum []byte, err error) {
	size := 4 * 1024 * 1024 //4M缓存
	if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
		if l.N < 1 {
			size = 1
		} else {
			size = int(l.N)
		}
	}
	md5Ctx := md5.New()
	sum := 0
	buf := make([]byte, size)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = errors.New("short write")
				break
			}
			md5Ctx.Write(buf[0:nr])
			if pbar != nil {
				pbar.Add(nr)
			}
		}
		sum += nr
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return md5Ctx.Sum(nil), err
}
