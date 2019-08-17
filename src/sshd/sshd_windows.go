package sshd

import (
	"fmt"
	"os"
)

func InitServer() {
	fmt.Fprintf(os.Stderr, "Windows can't support sshd\n")
}
