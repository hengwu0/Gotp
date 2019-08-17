package main

import (
	"bufio"
	"flag"
	log "github.com"
	"os"
	"protocol"
	"runtime"
	"sshd"
	"time"

	LOG "log"
	"net/http"
	_ "net/http/pprof"
)

var godaemon = flag.Bool("d", false, "Run app as a daemon with -d or -d=true.")
var gosshd = flag.Bool("sshd", false, "init sshd server together.")
var godaemon2 = flag.Bool("d2", false, "Create app with input off.")

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}
	if *godaemon && !*godaemon2 {
		daemon()
	}
	runtime.GOMAXPROCS(runtime.NumCPU()) //限制同时运行的goroutines数量
	go func() {
		LOG.Println(http.ListenAndServe("0.0.0.0:6068", nil))
	}()

	signal_process()

	protocol.Listen()
	if *gosshd {
		sshd.InitServer()
	}

	if *godaemon2 {
		for {
			time.Sleep(time.Second * 3600)
		}
	} else {
		line := protocol.CreateNew()
		defer line.Close()
		protocol.Readline_process(line, false)
	}

	log.Finish()
}

func Scanf(a *string) {
	reader := bufio.NewReader(os.Stdin)
	data, _, _ := reader.ReadLine()
	*a = string(data)
}
