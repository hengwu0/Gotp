package github

import (
	"files"
	"fmt"
	LOG "github.com/alecthomas/log4go"
	"io"
	"os"
	"time"
)

var (
	Log      LOG.Logger
	filename string
	console  string
)

func init() {
	if fpath, err := files.GetCurrentDirectory(); err == nil {
		filename = fpath + "/" + time.Now().Format("2006-01-02") + ".log"
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// Get a new logger instance
	Log = LOG.NewLogger()

	// Create a default logger that is logging messages of FINE or higher
	Log.AddFilter("file", LOG.FINE, LOG.NewFileLogWriter(filename, false))
	Log.Fine("==============================Program new start==============================")
	Log.Close()

	/* Can also specify manually via the following: (these are the defaults) */
	flw := LOG.NewFileLogWriter(filename, false)
	flw.SetFormat("[%D %T] [%L] %M")
	flw.SetRotate(false)
	flw.SetRotateSize(0)
	flw.SetRotateLines(0)
	flw.SetRotateDaily(false)
	Log.AddFilter("file", LOG.FINE, flw)

}

func Finish() {
	Log.Close()
	// Get a new logger instance
	Log = LOG.NewLogger()

	Log.AddFilter("file", LOG.FINE, LOG.NewFileLogWriter(filename, false))
	Log.Fine("=============================================================================")
	Log.Fine("")
	// bug：其通过channel传输给write协程写日志。当程序结束时，最后的日志可能会写入失败，时延不一定能解决问题
	time.Sleep(time.Millisecond * 10)

	Log.Close()
}

func Console(format string, a ...interface{}) {
	console += fmt.Sprintf(format, a...)
	//ConsoleFlush()	不能立即输出，因为在readline输出时，会使此处输出无效
}

func ConsoleError(format string, a ...interface{}) {
	console = console + "ERROR: " + fmt.Sprintf(format, a...)
	//ConsoleFlush()	不能立即输出，因为在readline输出时，会使此处输出无效
}

func ConsoleFlush(output io.Writer) {
	fmt.Fprintf(output, console)
	console = ""
}
