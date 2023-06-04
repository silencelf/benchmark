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
}

type counter struct {
	completed int
	success   int
	total     int
}

func NewRunner(host string, concurrency int, iteration int, headers Headers) *Runner {
	return &Runner{url: host, concurrency: concurrency, iteration: iteration, ct: counter{total: concurrency * iteration}, headers: headers}
}

func (r *Runner) Run() {
	for i := r.iteration; i > 0; i-- {
		log.Println("Iteration", r.iteration-i+1)
		for c := r.concurrency; c > 0; c-- {
			go r.request()
		}
		time.Sleep(time.Second)
	}
}

func (r *Runner) IncreaseCounter(success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ct.completed++
	if success {
		r.ct.success++
	}

	if r.ct.completed == r.ct.total {
		log.Printf("Success/Total: %v/%v. \n", r.ct.success, r.ct.total)
	}
}

func (r *Runner) request() {
	start := time.Now()
	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		fmt.Println(err.Error())
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
	}
	defer resp.Body.Close()
	defer r.IncreaseCounter(resp.StatusCode == http.StatusOK)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	end := time.Now()
	duration := end.Sub(start)
	log.Printf("Response Status: %v, Response Length: %v, Duration: %v", resp.Status, len(body), duration)
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
	c := flag.Int("c", 1, "concurrency level")
	i := flag.Int("i", 1, "iterations")
	var headers Headers
	flag.Var(&headers, "H", "")

	host := flag.String("h", "", "host url")
	flag.Parse()

	r := NewRunner(*host, *c, *i, headers)
	r.Run()

	<-quit
	fmt.Println("Shutdown...")
}
