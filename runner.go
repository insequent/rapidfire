package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	err   error
	group int
	resp  *http.Response
}

type Runner struct {
	abortChan  chan struct{}
	errChan    chan error
	results    map[int][]*Result
	resultChan chan *Result
	rLock      sync.Mutex
	rps        int
	stopChan   chan struct{}
	wg         *sync.WaitGroup
	verbose    bool
}

func NewRunner(verbose bool) *Runner {
	return &Runner{
		abortChan:  make(chan struct{}),
		resultChan: make(chan *Result),
		results:    map[int][]*Result{},
		stopChan:   make(chan struct{}),
		wg:         &sync.WaitGroup{},
		verbose:    verbose,
	}
}

func (r *Runner) Abort() {
	fmt.Println("Aborting everything!")
	close(r.abortChan)
}

func (r *Runner) Start(duration, rps int, method, url, data string) {
	r.rps = rps
	go r.collect()

	group := 1
	r.results = map[int][]*Result{}
	ticker := time.NewTicker(1 * time.Second)
	r.debug("Runner loop starting")
	for {
		select {
		case <-ticker.C:
			for i := 1; i <= rps; i++ {
				go r.request(group, method, url, data)
			}
			if duration > 0 && group >= duration {
				r.debug("Runner is stopping...")
				r.Stop()
				return
			}
			group++
		case <-r.abortChan:
			r.debug("Runner is aborting!")
			return
		}
	}
}

func (r *Runner) collect() {
	r.wg.Add(1)
	defer r.wg.Done()

	for {
		select {
		case result := <-r.resultChan:
			go r.processResult(result)
		case <-r.stopChan:
			r.debug("Collector is stopping...")
			return
		case <-r.abortChan:
			r.debug("Collector is aborting!")
			return
		}
	}
}

func (r *Runner) processResult(result *Result) {
	r.wg.Add(1)
	defer r.wg.Done()

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.results[result.group] = append(r.results[result.group], result)

	total := len(r.results[result.group])
	if total == r.rps {
		// TODO: This could be cleaner...
		codes := map[int]int{}
		errors := 0
		for _, res := range r.results[result.group] {
			if res.err != nil {
				r.debug(fmt.Sprintf("Error in response: %v", res.err))
				errors++
				continue
			}
			if _, ok := codes[res.resp.StatusCode]; ok {
				codes[res.resp.StatusCode]++
			} else {
				codes[res.resp.StatusCode] = 1
			}
		}

		codeStrs := ""
		for code, num := range codes {
			codeStrs += fmt.Sprintf(" Code %d: %d (%.0f%%)", code, num, float64(num)/float64(total)*100)
		}

		errPerc := float64(errors) / float64(total) * 100
		fmt.Printf("%d: Reqs: %d Errs: %d (%.0f%%)%s\n", result.group, total, errors, errPerc, codeStrs)
		return
	}
	r.debug(fmt.Sprintf("Count not met, waiting for more (count: %d, rps: %d)", total, r.rps))
}

func (r *Runner) request(group int, method, url, data string) {
	r.wg.Add(1)
	defer r.wg.Done()

	resp, err := http.Get(url)
	r.resultChan <- &Result{
		err:   err,
		group: group,
		resp:  resp,
	}

	// Ignoring body of response for now
	if err == nil {
		if e := resp.Body.Close(); e != nil {
			fmt.Printf("Error closing response body: %v", err)
		}
	}
}

func (r *Runner) debug(message string) {
	if r.verbose {
		fmt.Println(message)
	}
}

func (r *Runner) Stop() {
	close(r.stopChan)
	r.wg.Wait()
}
