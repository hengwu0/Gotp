// +build !windows

package sshd

import (
	log "github.com"
	"github.com/jpillora/sshd-lite/server"
)

func InitServer() {
	ssh_server, err := sshd.NewServer(&sshd.Config{
		Host:       "0.0.0.0",
		Port:       "2200",
		AuthType:   "wuheng:007",
		KeyFile:    "",
		LogVerbose: true,
	})
	if err != nil {
		log.ConsoleError("sshd init failed!...(err)\n", err)
	} else {
		go ssh_server.Start()
	}
}
