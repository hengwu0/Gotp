// +build !arm64
// +build !windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"sync"
)

var c = make(chan os.Signal)

func signal_process() {
	signal.Notify(c, syscall.SIGTTOU, syscall.SIGTTIN)
	go signalProcess()
}

//-d && SIGTTIN,SIGTTOU信号处理
func daemon() {
	fmt.Println("Switched to background!")
	//使用exec.Command方式无法在接收信号后执行命令
	pid, r2, sysErr := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)
	if sysErr != 0 || pid < 0 {
		fmt.Errorf("Fail to call fork")
		os.Exit(0)
	} else {
		ret, err := syscall.Setsid()
		if err != nil || ret < 0 {
			fmt.Errorf("Fail to call setsid")
			os.Exit(0)
		}
	}
	if pid == 0 || (r2 == 1 && runtime.GOOS == "darwin") {
		var zeroProcAttr syscall.ProcAttr
		args := os.Args
		args = append(args, "-d2")
		syscall.Exec(args[0], args, zeroProcAttr.Env)
		return
	}
	fmt.Printf("%s [%d] background running...\n", os.Args[0], pid)
	os.Exit(0)
}

func signalProcess() {
	var once sync.Once
	for {
		s := <-c
		switch s {
		case syscall.SIGTTIN, syscall.SIGTTOU:
			signal.Ignore(s)
			go once.Do(daemon)
		default:
			fmt.Printf("Recv signal:%#v\n", s)
		}
	}
}
