package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func main() {
	tmpFiles, err := os.MkdirTemp("", "myApp_*")
	if err != nil {
		log.Println("error creating temp file directory: ", err)
		os.Exit(1)
	}
	fmt.Println("temp files located at: ", tmpFiles)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	sigHandler, stopSig := newSignaling()
	sigHandler.register(
		func() {
			cleanup(cancel, wg, tmpFiles)
			os.Exit(1)
		},
		syscall.SIGINT, syscall.SIGTERM,
	)
	sigHandler.register(
		func() {
			cleanup(cancel, wg, tmpFiles)
			panic("SIGQUIT called")
		},
		syscall.SIGQUIT,
	)

	go func() {
		defer stopSig()
		defer wg.Done()
		createFiles(ctx, tmpFiles)
	}()

	sigHandler.handle()

	fmt.Println("Done")
}

func createFiles(ctx context.Context, tmpFiles string) {
	for i := 0; i < 30; i++ {
		if err := ctx.Err(); err != nil {
			return
		}
		_, err := os.Create(filepath.Join(tmpFiles, strconv.Itoa(i)))
		if err != nil {
			panic(err)
		}
		fmt.Println("Created file: ", i)
		time.Sleep(1 * time.Second)
	}
}

type signaling struct {
	ctx    context.Context
	notify chan os.Signal

	handlers map[os.Signal]func()
}

func newSignaling() (sig signaling, stop context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return signaling{
		ctx:      ctx,
		notify:   make(chan os.Signal, 1),
		handlers: map[os.Signal]func(){},
	}, cancel
}

func (s signaling) register(f func(), sigs ...os.Signal) {
	signal.Notify(
		s.notify,
		sigs...,
	)
	for _, sig := range sigs {
		s.handlers[sig] = f
	}
}

func (s signaling) handle() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case sig := <-s.notify:
			f, ok := s.handlers[sig]
			if !ok {
				log.Printf("unknown signal received: %v", sig)
				continue
			}
			f()
		}
	}
}

func cleanup(cancel context.CancelFunc, wg *sync.WaitGroup, tmpFiles string) {
	cancel()
	wg.Wait()

	if err := os.RemoveAll(tmpFiles); err != nil {
		fmt.Println("problem doing file cleanup: ", err)
		return
	}
	fmt.Println("cleanup done")
}
