package main

import (
	"os"
	"runtime/pprof"
	"time"
)

func init() {
	go func() {
		prof := pprof.Lookup("goroutine")
		tick := time.NewTicker(time.Second * 7)
		for range tick.C {
			os.Stderr.WriteString("-----\n")
			prof.WriteTo(os.Stderr, 1)
		}
	}()
}
