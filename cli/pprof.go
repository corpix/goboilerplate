package cli

import (
	"os"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/log"
)

// writeProfile writes cpu and heap profile into files.
func writeProfile(l log.Logger) error {
	cpu, err := os.Create("cpu.prof")
	if err != nil {
		return err
	}
	heap, err := os.Create("heap.prof")
	if err != nil {
		return err
	}

	pprof.StartCPUProfile(cpu)
	go func() {
		defer cpu.Close()
		defer heap.Close()

		l.Info().Msg("profiling, will exit in 30 seconds")
		time.Sleep(30 * time.Second)
		pprof.StopCPUProfile()
		pprof.WriteHeapProfile(heap)

		os.Exit(0)
	}()

	return nil
}

// writeTrace writes tracing data to file.
func writeTrace(l log.Logger) error {
	t, err := os.Create("trace.prof")
	if err != nil {
		return err
	}

	trace.Start(t)
	go func() {
		defer t.Close()

		l.Info().Msg("tracing, will exit in 30 seconds")
		time.Sleep(30 * time.Second)
		trace.Stop()

		os.Exit(0)
	}()

	return nil
}
