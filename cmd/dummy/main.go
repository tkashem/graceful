package main

import (
	"context"
	"flag"
	"sync"
	"time"

	"k8s.io/apiserver/pkg/server"
	"k8s.io/klog"
	"github.com/tkashem/graceful/pkg/core"
)

type Options struct {

}

var (
	concurrency = flag.Int("concurrency", 100, "number of concurrent workers")
	burst = flag.Int("burst", 10, "burst size")
	delay = flag.Duration("delay", 1 * time.Second, "step delay")
	duration = flag.Duration("duration", 1 * time.Minute, "test duration after steady state")
)

func main() {
	flag.Parse()

	shutdown, shutdownCancel := context.WithCancel(context.TODO())
	shutdownHandler := server.SetupSignalHandler()
	go func() {
		defer shutdownCancel()

		<-shutdownHandler
		klog.Info("Received SIGTERM or SIGINT signal, initiating shutdown.")
	}()

	// setup a dummy worker
	worker := func(wc *core.WorkerContext) {
		time.Sleep(time.Second)
	}

	// need this to wait for all workers to exit.
	wg := &sync.WaitGroup{}
	test, testCancel := context.WithTimeout(shutdown, *duration)
	defer testCancel()

	tc := core.TestContext{
		TestCancel: test,
		WaitGroup:  wg,
	}

	// run this worker in parallel
	runner := core.NewRunnerWithDelay(time.Millisecond)
	actions := runner.ToActions(&tc, *concurrency, worker, "dummy")
	generator := core.NewSteppedLoadGenerator(time.Second, 10)

	go generator.Generate(actions)

	klog.Info("waiting for test to complete")
	<-test.Done()

	klog.Info("waiting for worker(s) to be done")
	wg.Wait()

	klog.Info("all worker(s) are done")
}
