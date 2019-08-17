package protocol

import (
	"files"
	"fmt"
	log "github.com"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var port4gotp string = "6626"
var conns map[string]string = make(map[string]string)

func Listen() {
	RecvFile, _ = files.GetCurrentDirectory()
	listen, err := net.Listen("tcp", ":"+port4gotp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen ERROR: %s\n", err.Error())
		fmt.Fprintln(os.Stderr, "You should specified PORT or you can't recv cmds! But you can still send!")
		return
	}
	go Accept(listen)
}

func Write(conn net.Conn, date []byte) {
	conn.SetWriteDeadline(time.Now().Add(time.Duration(2) * time.Second))
	conn.Write(date)
	conn.SetWriteDeadline(time.Time{})
}

func Read(conn net.Conn, date []byte) {
	conn.SetReadDeadline(time.Now().Add(time.Duration(2) * time.Second))
	conn.Write(date)
	conn.SetReadDeadline(time.Time{})
}

func Trans(conn net.Conn, target string) {
	var conn_dest net.Conn
	var err error
	back := NewTPackHead()
	if conn_dest, err = build_send(target, os.Stderr); err != nil {
		log.Log.Info("conn--> %s Failed!(%s)", target, err.Error())
		conn.Write(back.EnpackHead('B', 404))
		return
	}
	conn.Write(back.EnpackHead('B', 240))

	close := func() {
		conn.Close()
		conn_dest.Close()
	}
	var once sync.Once
	go func() {
		io.Copy(conn_dest, conn)
		once.Do(close)
	}()
	go func() {
		io.Copy(conn, conn_dest)
		once.Do(close)
	}()
}

func Conn(cmds []string, output io.Writer) {
	if !strings.Contains(cmds[0], ":") {
		cmds[0] += ":" + port4gotp
	}
	if !strings.Contains(cmds[2], ":") {
		cmds[2] += ":" + port4gotp
	}
	if orig, ok := conns[cmds[0]]; ok && orig != cmds[2] {
		fmt.Fprintf(output, "Warning!!!\n")
		fmt.Fprintf(output, "orig: connect "+conns[cmds[0]]+" --> "+cmds[0]+"\n")
		fmt.Fprintf(output, "now : connect "+cmds[2]+" --> "+cmds[0]+"\n")
	}
	if cmds[0] == cmds[2] {
		fmt.Fprintf(output, "Can't connect "+conns[cmds[0]]+" --> "+cmds[0]+"\n")
		return
	}
	conns[cmds[0]] = cmds[2]
}

func Pnt() {
	fmt.Printf("###%#v\n", conns)
}
