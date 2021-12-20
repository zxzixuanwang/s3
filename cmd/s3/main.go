package main

import (
	"os"
	"runtime"

	"github.com/barnybug/s3"
)

func main() {
	runtime.GOMAXPROCS(2)
	exitCode := s3.Main(nil, os.Args, os.Stdout)
	os.Exit(exitCode)
}
