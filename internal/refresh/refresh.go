// Package refresh runs background catalog refreshes behind the UI: one pass
// shortly after startup, then every interval while the app stays open.
package refresh

import (
	"log"
	"sync"
	"time"
)

type job struct {
	name string
	run  func() error
}

// Interval is how often cached content is revalidated while the app is open.
const Interval = 4 * time.Hour

// settle delays the first pass so startup and UI loads get the network first.
const settle = 2 * time.Second

var (
	mu   sync.Mutex
	jobs []job
	once sync.Once
)

func Register(name string, run func() error) {
	mu.Lock()
	defer mu.Unlock()
	jobs = append(jobs, job{name: name, run: run})
}

// Start launches every registered job in the background. Notify runs after each
// completed pass so callers can tell the UI cached content was revalidated.
func Start(notify func()) {
	once.Do(func() {
		mu.Lock()
		holder := append([]job(nil), jobs...)
		mu.Unlock()
		for _, j := range holder {
			go loop(j, notify)
		}
	})
}

func loop(j job, notify func()) {
	time.Sleep(settle)
	for {
		started := time.Now()
		if err := j.run(); err != nil {
			log.Printf("refresh %s: %v", j.name, err)
		} else if notify != nil {
			notify()
		}
		wait := Interval - time.Since(started)
		if wait < 30*time.Second {
			wait = 30 * time.Second
		}
		time.Sleep(wait)
	}
}
