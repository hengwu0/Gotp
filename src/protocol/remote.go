package protocol

import (
	"github.com/chzyer/readline"
	"net"
)

func RecvRemote(conn net.Conn) {
	defer conn.Close()
	rl, err := readline.HandleConn(*CreateCfg(), conn)
	if err != nil {
		return
	}
	defer rl.Close()
	rl.SetPrompt("Remote(" + conn.LocalAddr().String() + "): ")
	Readline_process(rl, true)
}
