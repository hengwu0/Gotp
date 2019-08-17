package protocol

import (
	"files"
	"fmt"
	"github.com/chzyer/readline"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func CreateCfg() *readline.Config {
	return &readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		HistoryFile:     "/tmp/readline.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	}
}

func usage(w io.Writer) {
	io.WriteString(w, "commands:\n")
	io.WriteString(w, completer.Tree("    "))
}

func listFiles(path string) func(string) []string {
	return func(path string) []string {
		if len(path) == 0 || path == " " {
			path = "./"
		} else if path[len(path)-1] != '\\' && path[len(path)-1] != '/' {
			return []string{}
		}
		names := make([]string, 0)
		files, _ := ioutil.ReadDir(path)
		for _, f := range files {
			if f.IsDir() || f.Mode()&os.ModeSymlink != 0 { //软链接判断
				names = append(names, f.Name()+"/")
			} else {
				names = append(names, f.Name())
			}
		}
		if len(files) == 0 {
			names = append(names, "./")
			names = append(names, "../")
		}
		return names
	}
}

var completer = readline.NewPrefixCompleter(
	readline.PcItem("send"),
	readline.PcItem("conn"),
	readline.PcItem("remote"),
	readline.PcItem("GetRecvPath"),
	readline.PcItem("SetRecvPath"),
	readline.PcItem("help"),
	readline.PcItem("quit"),
	readline.PcItemDynamic(listFiles("../")),
)

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func CreateNew() *readline.Instance {
	line, _ := readline.NewEx(CreateCfg())
	return line
}

func Readline_process(l *readline.Instance, remote bool) {
	for {
		cmd, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(cmd) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		cmd = strings.TrimSpace(cmd)
		cmds := strings.Split(cmd, " ")

		switch cmds[0] {
		case "send", "s":
			cmds = cmds[1:]
			if len(cmds) < 2 {
				fmt.Fprintf(l.Stderr(), "usage: send IP[:PORT] file/dir1 [file/dir2] ... \n")
				continue
			}
			Send(cmds, l)
		case "conn", "c":
			cmds = cmds[1:]
			if len(cmds) < 3 || cmds[1] != "from" {
				fmt.Fprintf(l.Stderr(), "usage: conn IP[:PORT] from IP[:PORT]\n")
				continue
			}
			Conn(cmds, l.Stdout())
		case "remote", "r":
			if len(cmds) != 2 {
				fmt.Fprintf(l.Stderr(), "usage: remote IP[:PORT]\n")
				continue
			}
			if remote {
				fmt.Fprintf(l.Stderr(), "You can't use remote in remote mode!\n")
				continue
			}
			SendRemote(cmds[1], l.Stderr())
		case "help", "h":
			usage(l.Stderr())
			fmt.Fprintf(l.Stdout(), "usage: send IP[:PORT] file/dir1 [file/dir2] ... \n")
			fmt.Fprintf(l.Stdout(), "usage: conn IP[:PORT] from IP[:PORT]\n")
		case "quit", "q", "exit":
			return
		case "history":
			cmd = "cat -n " + l.Config.HistoryFile
			cmds = strings.Split(cmd, " ")
			if err := ExecTryShell(cmds); err == nil {
				ExecShell(l, cmds, false)
			} else {
				cmd = "cat " + l.Config.HistoryFile
				cmds = strings.Split(cmd, " ")
				if err := ExecTryShell(cmds); err == nil {
					ExecShell(l, cmds, false)
				} else {
					fmt.Fprintln(l.Stderr(), "Can't read history file: "+l.Config.HistoryFile)
				}
			}
		case "GetRecvPath":
			fmt.Fprintln(l.Stdout(), "RecvPath: "+RecvFile)
		case "SetRecvPath":
			if len(cmds) != 2 {
				fmt.Fprintf(l.Stderr(), "usage: SetRecvPath path.\n")
				continue
			}
			if fpath, err := files.RecvPathCheck(cmds[1]); err == nil {
				RecvFile = fpath
			} else {
				fmt.Fprintln(l.Stderr(), err)
			}
		case "cd":
			if len(cmds) < 2 {
				if home, err := files.Home(); err == nil {
					if err = os.Chdir(home); err != nil {
						fmt.Fprintln(l.Stderr(), err)
					}
				} else {
					fmt.Fprintln(l.Stderr(), err)
				}
			} else {
				if err := os.Chdir(cmds[1]); err != nil {
					fmt.Fprintln(l.Stderr(), err)
				}
			}
		default:
			ExecShell(l, cmds, remote)
		case "":
		}
	}
}

// 仅可使用非阻塞的shell命令
func ExecTryShell(cmds []string) error {
	cmd := exec.Command(cmds[0])
	cmd.Args = cmds
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func ExecShell(l *readline.Instance, cmds []string, remote bool) {
	cmd := exec.Command(cmds[0])
	cmd.Args = cmds

	cmd.Stderr = l.Stderr()
	cmd.Stdout = l.Stdout()
	if remote {
		cmd.Stdin = nil
	} else {
		cmd.Stdin = os.Stdin
	}
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(l.Stderr(), "cmd.Run: ", err)
		return
	}
}
