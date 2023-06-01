package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type runner struct {
	url         string
	concurrency int
	iteration   int
}

func (r *runner) run() {
	for i := r.iteration; i > 0; i-- {
		log.Println("Iteration", r.iteration-i+1)
		for c := r.concurrency; c > 0; c-- {
			go func() {
				resp, err := http.Get(r.url)
				if err != nil {
					fmt.Println(err.Error())
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Println(err.Error())
				}
				log.Println("Response Status:", resp.Status)
				log.Println("Response Length:", len(body))
			}()
		}
		time.Sleep(time.Second)
	}
}

func main() {
	//ch := make(chan int)
	c := flag.Int("c", 1, "concurrency level")
	i := flag.Int("i", 1, "iterations")
	host := flag.String("h", "http://baidu.com", "host url")
	flag.Parse()

	r := runner{url: *host, concurrency: *c, iteration: *i}
	r.run()

	//fmt.Println("Ctl+C to exit...")
	//<-ch
}
