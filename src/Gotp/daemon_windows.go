package main

import (
	"fmt"
	"os"
)

func signal_process() {
}

//-d信号处理
func daemon() {
	fmt.Fprintf(os.Stderr, "Windows can't support -d flag\n")
}
