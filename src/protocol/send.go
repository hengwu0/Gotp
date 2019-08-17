package protocol

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"files"
	log "github.com"
	"github.com/chzyer/readline"
	"gopkg.in/cheggaaa/pb.v1"

	"io"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

func SendRemote(addr string, output io.Writer) {
	var conn net.Conn
	var err error
	if conn, err = SendHead(addr, 0, output); err != nil {
		return
	}
	defer func() {
		conn.Close()
		if p := recover(); p != nil {
			debug.PrintStack()
			log.ConsoleError("SendRemote failed(%s)!?\n", addr)
			log.ConsoleFlush(output)
			return
		}
	}()

	pack := NewTPack()
	conn.SetWriteDeadline(time.Now().Add(time.Duration(2) * time.Second))
	conn.Write(pack.Enpack('R', 0, ""))
	conn.SetWriteDeadline(time.Time{})
	if res := SendBack(conn); res != 220 {
		log.ConsoleError("sendback err(%d)", res)
		log.ConsoleFlush(output)
		return
	}

	cli, err := readline.NewRemoteCli(conn)
	if err != nil {
		log.ConsoleError("readline send err", err)
		log.ConsoleFlush(output)
		return
	}
	log.Console("This remote mode doesn't support Commands with interaction!\n")
	log.ConsoleFlush(output)
	cli.Serve()
}

func SendHead(addr string, multi uint64, output io.Writer) (net.Conn, error) {
	var conn net.Conn
	var err error
	if conn, err = build_send(addr, output); err != nil {
		log.ConsoleError("build_send1 error(%s)!\n", addr)
		log.ConsoleFlush(output)
		return nil, err
	}
	head := NewTPackHead()
	conn.SetWriteDeadline(time.Now().Add(time.Duration(2) * time.Second))
	conn.Write(head.EnpackHead('H', multi))
	conn.SetWriteDeadline(time.Time{})
	if SendBack(conn) != 200 {
		log.ConsoleFlush(output)
		conn.Close()
		return nil, errors.New("recv code err!")
	}
	if multi != 0 {
		conn.Close()
		return nil, nil
	}
	return conn, nil
}

func Send(cmds []string, l *readline.Instance) {
	var conn net.Conn
	var err error
	if _, err = SendHead(cmds[0], uint64(len(cmds)-1), l.Stderr()); err != nil {
		return
	}
	pool, _ := pb.StartPool()
	pool.Output = l.Stdout()
	wg := new(sync.WaitGroup)

	for _, filename := range cmds[1:] {
		if conn, err = build_send(cmds[0], l.Stderr()); err != nil {
			log.ConsoleError("build_send2 error!\n")
			log.ConsoleFlush(l.Stderr())
			return
		}
		if filename == "" {
			log.ConsoleFlush(l.Stderr())
			conn.Close()
			continue
		}
		wg.Add(1)
		go SendTar(wg, filename, conn, pool)
	}
	wg.Wait()
	pool.Stop()
	log.ConsoleFlush(l.Stderr())
}

func build_send(target string, output io.Writer) (conn net.Conn, err error) {
	var orig string
	var ok bool
	if !strings.Contains(target, ":") {
		target += ":" + port4gotp
	}

	if orig, ok = conns[target]; ok {
		conn, err = net.DialTimeout("tcp", orig, time.Second+time.Second/2)
	} else {
		conn, err = net.DialTimeout("tcp", target, time.Second+time.Second/2)
	}

	if err != nil {
		log.ConsoleError("Connecting Failed: %s\n", err.Error())
		return
	}

	// Trans mode
	if ok {
		head := NewTPackHead()
		Write(conn, head.EnpackHead('H', 0))
		if SendBack(conn) != 200 {
			conn.Close()
			log.ConsoleError("Connect %s --> %s error!\n", orig, target)
			log.ConsoleFlush(output)
			err = errors.New("Connect Failed!")
			return
		}
		//send ip:port and recv status
		head2 := NewTPack()
		conn.SetWriteDeadline(time.Now().Add(time.Duration(2) * time.Second))
		conn.Write(head2.Enpack('T', 0, target))
		conn.SetWriteDeadline(time.Time{})
		if SendBack(conn) != 240 {
			conn.Close()
			log.ConsoleError("Connect2 %s --> %s error!\n", orig, target)
			log.ConsoleFlush(output)
			err = errors.New("Connect2 Failed!")
			return
		}
	}
	return
}

type TarGz struct {
	conn net.Conn
	tw   *tar.Writer
	gw   *gzip.Writer
	pbar *pb.ProgressBar
}

func SendTar(wg *sync.WaitGroup, filesName string, conn net.Conn, pool *pb.Pool) {
	defer func() {
		wg.Done()
		conn.Close()
		if p := recover(); p != nil {
			debug.PrintStack()
			log.ConsoleError("Send file failed! Not enouth room for %s?\n", filesName)
			return
		}
	}()

	fileName := files.Filename(filesName)
	fileSize := files.FileSize(filesName)
	pack := NewTPack()
	conn.SetWriteDeadline(time.Now().Add(time.Duration(2) * time.Second))
	conn.Write(pack.Enpack('F', fileSize, fileName))
	conn.SetWriteDeadline(time.Time{})
	if SendBack(conn) != 210 {
		return
	}

	myTar := &TarGz{}
	myTar.pbar = pb.New64(fileSize).Prefix(fileName)
	pool.Add(myTar.pbar)
	myTar.conn = conn

	myTar.sendTarFile(filesName, true)
	myTar.pbar.Finish()
}

//更多可以参考http://lib.csdn.net/article/go/68111
func (myTar *TarGz) gzCompress(file *os.File, prefix string) {
	info, err := file.Stat()
	if err != nil {
		log.ConsoleError("Stat \"%s\" err: %s!\n", file.Name(), err.Error())
		return
	}
	if info.IsDir() {
		// 信息头
		header, err := tar.FileInfoHeader(info, "")
		header.Name = prefix + "/" + header.Name
		err = myTar.tw.WriteHeader(header)
		if err != nil {
			log.ConsoleError("Write tar head for dir err: %s! Path=%s\n", err.Error(), file.Name())
			return
		}

		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			log.ConsoleError("Readdir : %s!\n", err.Error())
			panic(nil)
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				log.ConsoleError("Open: %s!\n", err.Error())
				continue
			}
			myTar.gzCompress(f, prefix)
			f.Close()
		}
	} else {
		// 信息头
		header, _ := tar.FileInfoHeader(info, "")
		header.Name = prefix + "/" + header.Name
		err := myTar.tw.WriteHeader(header)
		if err != nil {
			log.ConsoleError("Write tar head for file err: %s! File=%s\n", err.Error(), file.Name())
			panic(nil)
		}
		md5sum, err := files.Copy(myTar.tw, file, myTar.pbar)
		if err != nil {
			log.ConsoleError("IO copy err, No enouth room?: %s! File=%s\n", err.Error(), file.Name())
			panic(nil)
		}
		myTar.tw.Flush()
		myTar.gw.Flush()
		//recv md5 and check
		if myTar.SendTarback(md5sum) == false {
			log.ConsoleError("Check md5(%x) for %s(send-ed) failed! Please resend!\n", md5sum, file.Name())
			return
		}
	}
	return
}

func (myTar *TarGz) sendTarFile(filesName string, singleFile bool) {
	// gzip write
	myTar.gw = gzip.NewWriter(myTar.conn)
	defer myTar.gw.Close()
	// tar write
	myTar.tw = tar.NewWriter(myTar.gw)
	defer myTar.tw.Close()
	// 打开文件(夹)
	dir, err := os.Open(filesName)
	if err != nil {
		log.ConsoleError("Open: %s!\n", err.Error())
		return
	}
	defer dir.Close()
	myTar.gzCompress(dir, "")
}

func SendBack(conn net.Conn) int {
	back := NewTPackHead()
	conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	_, err := io.ReadFull(conn, back.buf)
	if err != nil || len(back.buf) == 0 {
		log.ConsoleError("conn SendBack error: %s\n", err.Error())
		//debug.PrintStack()
		return 0
	}
	conn.SetReadDeadline(time.Time{})
	if f, stat := back.DepackHead(); f == 'B' {
		return int(stat)
	} else {
		log.ConsoleError("conn SendBack flag:%c, stat:%d\n", f, stat)
	}
	return 0
}

func (myTar *TarGz) SendTarback(send []byte) bool {
	back := NewTMd5Pack()
	myTar.conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	_, err := io.ReadFull(myTar.conn, back.buf)
	if err != nil || len(back.buf) == 0 {
		log.ConsoleError("Recv Md5sum error: %s\n", err.Error())
		//debug.PrintStack()
		return false
	}
	myTar.conn.SetReadDeadline(time.Time{})
	if flag, recv := back.DepackMd5sum(); flag == 'C' {
		return checkMd5(recv, send)
	}
	return false
}
