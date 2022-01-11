package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Result is a collection of data related to the result of a single http request
type Result struct {
	err   error
	group int
	resp  *http.Response
}

// Runner is the organizational struct for all request orchestration and handling
type Runner struct {
	abortChan   chan struct{}         // Channel to signal abort of all actions
	burst       bool                  // Enables/disables burst control
	burstChans  map[int]chan struct{} // Map of channels for signaling HTTP request go routines to start by group id
	results     map[int][]*Result     // Map of http request results
	resultChan  chan *Result          // Channel for sending request data from clients to Runner's collector
	resultsLock sync.Mutex            // Lock for Runner.results
	rps         int                   // Requests per second
	wg          *sync.WaitGroup       // Primary waitgroup
	verbose     bool                  // Verbose logging
}

// NewRunner is a handy constructor for Runner
func NewRunner(verbose bool) *Runner {
	return &Runner{
		abortChan:  make(chan struct{}),
		burstChans: map[int]chan struct{}{},
		resultChan: make(chan *Result),
		results:    map[int][]*Result{},
		wg:         &sync.WaitGroup{},
		verbose:    verbose,
	}
}

// Abort is a signal to all goroutines to stop ASAP
func (r *Runner) Abort() {
	fmt.Println("Aborting everything!")
	close(r.abortChan)
}

// debug logs to terminal if the verbose flag is passed
func (r *Runner) debug(message string) {
	if r.verbose {
		fmt.Println(message)
	}
}

// Start begins the per second request cycle
func (r *Runner) Start(burst bool, duration, rps int, method, url, data string) {
	r.burst = burst
	r.rps = rps

	group := 1
	ticker := time.NewTicker(1 * time.Second)
	r.debug("Runner loop starting")
	for {
		select {
		case <-ticker.C:
			// TODO: Need to test if burst actually makes a difference
			r.debug(fmt.Sprintf("Starting requests for group %d", group))
			if burst {
				r.burstChans[group] = make(chan struct{})
			}
			for i := 1; i <= rps; i++ {
				// Adding to waitgroup here to ensure inclusion in wait below
				r.wg.Add(1)
				go r.request(group, method, url, data)
			}
			if burst {
				close(r.burstChans[group])
			}

			if duration > 0 && group >= duration {
				r.debug("Runner is stopping...")
				r.wg.Wait()
				return
			}
			group++
		case <-r.abortChan:
			r.debug("Runner is aborting!")
			return
		}
	}
}

// outputResults prints the group results to terminal and cleans up unneeded group data
func (r *Runner) outputResults(group, total int) {
	defer r.wg.Done()

	// TODO: This could be cleaner...
	r.resultsLock.Lock()
	codes := map[int]int{}
	errors := 0
	for _, res := range r.results[group] {
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
	r.resultsLock.Unlock()

	codeStrs := ""
	for code, num := range codes {
		codeStrs += fmt.Sprintf(" %d: %d (%.0f%%)", code, num, float64(num)/float64(total)*100)
	}

	errPerc := float64(errors) / float64(total) * 100
	fmt.Printf("%d: Reqs: %d Errs: %d (%.0f%%)%s\n", group, total, errors, errPerc, codeStrs)
}

// processResult takes an individual result, stores it, and if it is the last of the group, calls outputGroup
// This is intended to be its own go routine
func (r *Runner) processResult(result *Result) {
	defer r.wg.Done()

	r.resultsLock.Lock()
	defer r.resultsLock.Unlock()
	r.results[result.group] = append(r.results[result.group], result)

	total := len(r.results[result.group])
	if total == r.rps {
		r.wg.Add(1)
		go r.outputResults(result.group, total)
		return
	}
	r.debug(fmt.Sprintf("Count not met, waiting for more (count: %d, rps: %d)", total, r.rps))
}

// request makes a single HTTP request
// This is intended to be its own go routine
func (r *Runner) request(group int, method, url, data string) {
	defer r.wg.Done()

	if r.burst {
		<-r.burstChans[group]
	}

	var req *http.Request
	var resp *http.Response
	var err error
	switch strings.ToUpper(method) {
	case "DELETE":
		req, err = http.NewRequest("DELETE", url, nil)
	case "GET":
		req, err = http.NewRequest("GET", url, nil)
	case "HEAD":
		req, err = http.NewRequest("HEAD", url, nil)
	case "PATCH":
		buf := bytes.NewBufferString(data)
		req, err = http.NewRequest("PATCH", url, buf)
	case "POST":
		buf := bytes.NewBufferString(data)
		req, err = http.NewRequest("POST", url, buf)
	case "PUT":
		buf := bytes.NewBufferString(data)
		req, err = http.NewRequest("PUT", url, buf)
	}
	if err == nil {
		resp, err = http.DefaultClient.Do(req)
	}

	// Ignoring body of response for now
	if err == nil {
		// Close() helps ensure we don't reach open file limits too quickly
		if e := resp.Body.Close(); e != nil {
			fmt.Printf("Error closing response body: %v", err)
		}
	}

	r.wg.Add(1)
	go r.processResult(&Result{
		err:   err,
		group: group,
		resp:  resp,
	})
}
