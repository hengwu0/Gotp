package protocol

import (
	"files"
	log "github.com"
	"io"
	"net"
	"os"
	//"runtime/debug"
	"archive/tar"
	"compress/gzip"
	"time"
)

var RecvFile string

func Accept(listen net.Listener) {
	var count int = 0
	var flag byte
	head := NewTPackHead()
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Log.Error("accept error: %s", err.Error())
			log.Log.Error("You should change PORT or you can't recv cmds! But you can still send!")
			continue
		}

		//count用于跳过n个head
		if count == 0 {
			flag, count = head.RecvHead(conn)
			switch flag {
			case 'H':
				// count==0表示不立即断开socket
				if count == 0 {
					go RecvAll(conn)
				} else {
					conn.Close()
				}
			case 0:
				count = 0
				conn.Close()
			default:
				log.Log.Error("Gotp error: flag=%c,count=%d", flag, count)
			}
			continue
		}
		count--
		go RecvAll(conn)
	}
}

//dispatcher
//禁止创建协程
//不回收资源
func RecvAll(conn net.Conn) {
	var recv_ok bool = true
	pack := NewTPack()

	flag, _, path := RecvCmds(conn, pack)
	switch flag {
	case 'F':
		recv_ok = RecvTarFile(conn)
	case 'T':
		Trans(conn, path)
	case 'R':
		RecvRemote(conn)
	default:
		log.Log.Error("Recv unknow cmds error: cmd=%c", flag)
		return
	}
	if recv_ok == false {
		os.Remove(path)
		return
	}
}

func RecvTarFile(conn net.Conn) (flag bool) {
	defer conn.Close()
	var md5sum []byte
	// gzip read
	gr, err := gzip.NewReader(conn)
	if err != nil {
		log.Log.Error("Socket break before recving: " + err.Error())
		return
	}
	defer gr.Close()
	// tar read
	tr := tar.NewReader(gr)
	var name string
	// 读取文件
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Log.Error("TarGz broken while recving: " + err.Error())
			return false
		}

		name = RecvFile + h.Name
		// 显示文件
		if h.Typeflag == tar.TypeDir {
			if err = os.Mkdir(name, os.FileMode(h.Mode%512)); err != nil && !os.IsExist(err) {
				log.Log.Error("Create dir failed! limits of authority???" + err.Error())
				return false
			}
			continue
		}
		fw, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode%512))
		if err != nil && (files.IsBusy(err) || os.IsExist(err)) {
			os.Remove(name)
			fw, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY, os.FileMode(h.Mode%512))
			if err != nil {
				log.Log.Error("ReCreate file failed! " + err.Error())
				return false
			}
		} else if err != nil {
			log.Log.Error("Create file failed! " + err.Error())
			return false
		}
		// 写文件
		md5sum, err = files.Copy(fw, tr, nil)
		if err != nil {
			log.Log.Error("Write file failed! " + err.Error())
			fw.Close()
			return false
		}
		fw.Close()
		log.Log.Info("File recvd: %s, md5sum:%x", name, md5sum)
		RecvTarback(conn, md5sum)
	}

	return true
}

func checkMd5(recv []byte, send []byte) bool {
	for i := 0; i < 16; i++ {
		if recv[i] != send[i] {
			return false
		}
	}
	return true
}

func (packet *TPackHead) RecvHead(conn net.Conn) (byte, int) {
	conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	_, err := io.ReadFull(conn, packet.buf)
	if err != nil {
		log.Log.Error("read head error: ", err.Error())
		return 0, 0
	}
	conn.SetReadDeadline(time.Time{})
	head, count := packet.DepackHead()
	log.Log.Info("recv head: head=%c, count=%v", head, count)
	conn.Write(packet.EnpackHead('B', 200))
	return head, int(count)
}

func RecvCmds(conn net.Conn, pack *TPack) (flag byte, size int64, path string) {
	flag = 'Z'
	conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	_, err := io.ReadFull(conn, pack.buf)
	if err != nil {
		log.Log.Info("recv cmd finish, socket break!(%s)", err.Error())
		return
	}
	conn.SetReadDeadline(time.Time{})
	flag, size, path = pack.Depack()
	back := NewTPackHead()
	switch flag {
	case 'F':
		conn.Write(back.EnpackHead('B', 210))
	case 'T':
		//此处需待后一级反馈，不在此back
	case 'R':
		conn.Write(back.EnpackHead('B', 220))
	default:
		conn.Write(back.EnpackHead('B', 404))
	}
	log.Log.Info("recv cmd: flag=%c, size=%v, path=%s", flag, size, path)
	return
}

func Recvback(conn net.Conn) int {
	back := NewTPackHead()
	conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	_, err := io.ReadFull(conn, back.buf)
	if err != nil || len(back.buf) == 0 {
		log.ConsoleError("conn Recvback error: %s\n", err.Error())
		//debug.PrintStack()
		return 0
	}
	conn.SetReadDeadline(time.Time{})
	if f, stat := back.DepackHead(); f == 'H' {
		return int(stat)
	}
	return 0
}

func RecvTarback(conn net.Conn, recv []byte) {
	md5sum := NewTMd5Pack()
	conn.Write(md5sum.EnpackMd5sum('C', recv))
}
