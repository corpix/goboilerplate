package main

import (
	"runtime"

	"git.backbone/corpix/goboilerplate/cli"
)

func init() { runtime.GOMAXPROCS(runtime.NumCPU()) }
func main() { cli.Run() }
