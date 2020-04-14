package test

import (
	"sync"
	"time"
	"context"

	"k8s.io/klog"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type Initializer func() error
func (i Initializer) Initialize() error {
	return i()
}

type InitializerChain []Initializer
func (c InitializerChain) Invoke() error {
	allErrs := make([]error, 0)

	for _, initializer := range c {
		if err := initializer.Initialize(); err != nil {
			allErrs = append(allErrs, err)
			break
		}
	}

	return utilerrors.NewAggregate(allErrs)
}

type Worker func()
func (w Worker) Work() {
	w()
}

type WorkerConfig struct {
	Name string
	WaitInterval time.Duration
	Worker Worker
}

type WorkerChain []*WorkerConfig
func (c WorkerChain) Invoke(parent context.Context, wait *sync.WaitGroup) {
	withJitter := func(parent context.Context, wait *sync.WaitGroup, config *WorkerConfig) {
		<-time.After(utilwait.Jitter(time.Second, 5.0))
		go run(parent, wait, config)
	}

	for _, config := range c {
		go withJitter(parent, wait, config)
	}
}

func run(parent context.Context, wg *sync.WaitGroup, config *WorkerConfig) {
	wg.Add(1)
	defer wg.Done()

	klog.Infof("loop=%s - starting worker loop", config.Name)

	for {
		select {
		case <-parent.Done():
			klog.Infof("loop=%s - ending worker loop", config.Name)
			return
		case <-time.After(config.WaitInterval):
			config.Worker.Work()
		}
	}
}
