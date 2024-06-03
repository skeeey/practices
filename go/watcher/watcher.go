package main

import (
	"context"
	"fmt"
	"skeeey/go-test/signal"
	"sync"
	"time"
)

type Event struct {
	Type string
}

type Watcher struct {
	sync.Mutex

	result chan Event
	done   chan struct{}
}

func NewManifestWorkWatcher() *Watcher {
	mw := &Watcher{
		result: make(chan Event),
		done:   make(chan struct{}),
	}

	return mw
}

// ResultChan implements Interface.
func (mw *Watcher) ResultChan() <-chan Event {
	return mw.result
}

// Stop implements Interface.
func (mw *Watcher) Stop() {
	// Call Close() exactly once by locking and setting a flag.
	mw.Lock()
	defer mw.Unlock()
	// closing a closed channel always panics, therefore check before closing
	select {
	case <-mw.done:
		close(mw.result)
	default:
		close(mw.done)
	}
}

func (mw *Watcher) Receive(evt Event) {
	fmt.Printf("Blocking the event %v\n", evt.Type)

	mw.result <- evt

	fmt.Printf("Unblock the event %v\n", evt.Type)
}

func main() {
	shutdownCtx, cancel := context.WithCancel(context.TODO())
	shutdownHandler := signal.SetupSignalHandler()
	go func() {
		defer cancel()
		<-shutdownHandler
		fmt.Println("Received SIGTERM or SIGINT signal, shutting down controller.")
	}()

	ctx, terminate := context.WithCancel(shutdownCtx)
	defer terminate()

	w := NewManifestWorkWatcher()

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println("send event ", i)
			go w.Receive(Event{Type: fmt.Sprintf("test%d", i)})
			fmt.Println("send event ", i, " end")
		}
	}()

	time.Sleep(15 * time.Second)

	ch := w.ResultChan()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println(event)

		}
	}
}
