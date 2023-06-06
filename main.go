package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type Runner struct {
	url         string
	concurrency int
	iteration   int
	mu          sync.Mutex
	headers     Headers
	ct          counter
	results     []Result
	verbose     bool
}

type counter struct {
	completed int
	success   int
	total     int
}

type Result struct {
	url           string
	success       bool
	status        string
	statusCode    int
	contentLength int
	duration      time.Duration
}

func (r Result) String() string {
	return fmt.Sprint(r.url, r.status, r.statusCode, r.contentLength, r.duration.Milliseconds())
}

func NewRunner(host string, concurrency int, iteration int, headers Headers, verbose bool) *Runner {
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return &Runner{
		url:         host,
		concurrency: concurrency,
		iteration:   iteration,
		ct:          counter{total: concurrency * iteration},
		headers:     headers,
		verbose:     verbose,
	}
}

func (r *Runner) Run() {
	for i := r.iteration; i > 0; i-- {
		log.Println("Iteration", r.iteration-i+1)
		for c := r.concurrency; c > 0; c-- {
			go r.trackIt(r.request)
		}
		time.Sleep(time.Second)
	}
}

func (r *Runner) IncreaseCounter(re Result) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ct.completed++
	if re.success {
		r.ct.success++
	}
	r.results = append(r.results, re)

	if r.ct.completed == r.ct.total {
		for i, v := range r.results {
			fmt.Println(i, ":", v)
		}
		log.Printf("Success/Total: %v/%v. \n", r.ct.success, r.ct.total)
	}
}

func (r *Runner) request() (Result, error) {
	start := time.Now()
	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return Result{url: r.url, success: false, statusCode: -1, status: err.Error()}, err
	}

	for _, h := range r.headers {
		kv := strings.Split(h, ":")
		if len(kv) > 1 {
			req.Header.Add(kv[0], kv[1])
		}
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return Result{url: r.url, success: false, statusCode: -1, status: err.Error()}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return Result{url: r.url, success: false, statusCode: -1, status: err.Error()}, err
	}
	end := time.Now()
	duration := end.Sub(start)

	return Result{url: r.url, success: true, statusCode: resp.StatusCode, duration: duration, contentLength: len(body)}, nil
}

func (r *Runner) trackIt(fn func() (Result, error)) {
	start := time.Now()
	result, _ := fn()
	end := time.Now()
	result.duration = end.Sub(start)
	if r.verbose {
		log.Println(result)
	}
	r.IncreaseCounter(result)
}

type Headers []string

func (h *Headers) String() string {
	return strings.Join(*h, " ")
}

func (h *Headers) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func main() {
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	c := flag.Int("c", 1, "Concurrency level")
	i := flag.Int("i", 1, "Iterations")
	v := flag.Bool("v", false, "Verbose mode.")
	host := flag.String("h", "", "host url")
	var headers Headers
	flag.Var(&headers, "H", "")
	flag.Parse()

	r := NewRunner(*host, *c, *i, headers, *v)
	r.Run()

	<-quit

	fmt.Println("Shutdown...")
}
