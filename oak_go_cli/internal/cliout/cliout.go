package cliout

import (
	"fmt"
	"os"
)

func Infof(format string, v ...any) {
	fmt.Printf(format+"\n", v...)
}

func Warnf(format string, v ...any) {
	fmt.Printf(Yellow("WARN: ")+format+"\n", v...)
}

func Errorf(format string, v ...any) {
	_, _ = fmt.Fprintf(os.Stderr, Red("ERROR: ")+format+"\n", v...)
}
