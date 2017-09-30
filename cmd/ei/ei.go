package main

import (
	"../../../ei"
	"github.com/comail/colog"
	"log"
	"os"
)

const (
	Version  = "v0.0.1"
	progname = "ei"
)

func main() {
	colog.SetDefaultLevel(colog.LDebug)
	colog.SetMinLevel(colog.LInfo)
	colog.SetFormatter(&colog.StdFormatter{
		Colors: true,
		Flag:   log.Ldate | log.Ltime,
	})
	colog.Register()

	cli := &ei.CLI{
		OutStream: os.Stdout,
		ErrStream: os.Stderr,
		Version:   Version,
		Name:      progname,
	}
	os.Exit(cli.Run(os.Args))
}
