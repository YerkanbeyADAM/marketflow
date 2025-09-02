package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	Port        = flag.Int("port", 8080, "")
	Help        = flag.Bool("help", false, "")
	portFlagSet bool
)

func FlagHelp() {
	fmt.Println(`Simple Storage Service.

Usage:
    marketflow [-port <N>]
    marketflow --help

Options:
    --help     Show this screen.
    --port N   Port number`)
}

func ErrorHandling() {
	if *Port < 1024 || *Port > 49151 {
		log.Fatal("Port out of range")
		os.Exit(1)
	}
}

func InitFlags() {
	flag.Parse()
	flag.Usage = FlagHelp
	if *Help {
		flag.Usage()
		os.Exit(0)
	}
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portFlagSet = true
		}
	})
	ErrorHandling()
}

func PortFlagSet() bool {
	return portFlagSet
}
