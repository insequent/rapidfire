package main

import (
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Println("Usage: rp [options...] <url>")
	flag.PrintDefaults()
}

// Main
func main() {
	// Overwrite flag's Usage with our custom usage to include url arg
	flag.Usage = usage

	burst := flag.Bool("b", false, "Burst mode. Sends all requests per seconds as close to the same time as possible")
	duration := flag.Int("t", 0, "Length of time (in seconds) to run the queries. 0 for indefinite")
	method := flag.String("X", "GET", "The HTTP method to use with the request")
	payload := flag.String("d", "", "Data (Body) to send with the request")
	rps := flag.Int("r", 1, "Number of request to make each second")
	verbose := flag.Bool("v", false, "Enables verbose logging")
	flag.Parse()

	url := flag.Arg(0)
	if flag.NArg() == 0 || url == "" {
		fmt.Println("ERROR: A url is required")
		os.Exit(1)
	}

	runner := NewRunner(*verbose)
	runner.Start(*burst, *duration, *rps, *method, url, *payload)
}
