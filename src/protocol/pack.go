//协议：
// T: trans
// F: files	TPackHead
package protocol

import (
	"encoding/binary"
	"fmt"
	"os"
	//"runtime/debug"
)

const (
	constHeaderLength = 1
	constFlagLength   = 8
	ConstMaxLength    = 64 - 9
	constMd5Length    = 16
	PackheadSize      = constHeaderLength + constFlagLength
	PackSize          = constHeaderLength + constFlagLength + ConstMaxLength
	PackMd5sumSize    = constHeaderLength + constMd5Length
)

type TPack struct {
	buf []byte
}

func NewTPack() *TPack {
	packet := make([]byte, PackSize)
	return &TPack{packet}
}

type TPackHead struct {
	buf []byte
}

func NewTPackHead() *TPackHead {
	packet := make([]byte, PackheadSize)
	return &TPackHead{packet}
}

func (packet *TPackHead) EnpackHead(head byte, flag uint64) []byte {
	packet.buf[0] = head
	binary.BigEndian.PutUint64(packet.buf[constHeaderLength:PackheadSize], uint64(flag))
	return packet.buf
}
func (buffer *TPackHead) DepackHead() (head byte, flag uint64) {
	head = buffer.buf[0]
	flag = binary.BigEndian.Uint64(buffer.buf[constHeaderLength:PackheadSize])
	return
}

//封包
func (packet *TPack) Enpack(head byte, flag int64, path string) (buf []byte) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintf(os.Stderr, "path too long: %s?\n", path)
			buf = make([]byte, PackSize)
			buf[0] = 'X'
			//debug.PrintStack()
		}
	}()
	packet.buf[0] = head
	binary.BigEndian.PutUint64(packet.buf[constHeaderLength:PackheadSize], uint64(flag))
	copy(packet.buf[PackheadSize:], []byte(path))
	return packet.buf
}

//解包
func (buffer *TPack) Depack() (head byte, flag int64, path string) {
	head = buffer.buf[0]
	switch buffer.buf[0] {
	case 'F', 'T':
		flag = int64(binary.BigEndian.Uint64(buffer.buf[constHeaderLength:PackheadSize]))
		path = byteString(buffer.buf[PackheadSize:PackSize])
	case 'B':
		fallthrough
	default:
		flag = 0
		path = ""
	}
	return
}

type TMd5Pack struct {
	buf []byte
}

func NewTMd5Pack() *TMd5Pack {
	packet := make([]byte, PackMd5sumSize)
	return &TMd5Pack{packet}
}

func (packet *TMd5Pack) EnpackMd5sum(head byte, md5sum []byte) []byte {
	packet.buf[0] = head
	copy(packet.buf[constHeaderLength:], md5sum)
	return packet.buf
}

func (buffer *TMd5Pack) DepackMd5sum() (flag byte, md5sum []byte) {
	flag = buffer.buf[0]
	md5sum = buffer.buf[constHeaderLength:PackMd5sumSize]
	return
}

func byteString(p []byte) string {
	for i := 0; i < len(p); i++ {
		if p[i] == 0 {
			return string(p[0:i])
		}
	}
	return string(p)
}
