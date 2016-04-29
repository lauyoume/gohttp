package main

import (
	"log"
	"sync"
	"time"

	"github.com/lauyoume/gohttp"
)

func main() {
	gohttp.SetOption(&gohttp.Option{
		Address: []string{"104.238.193.74", "104.238.193.75"},
	})

	//runtime.GOMAXPROCS(10)
	req := gohttp.New()

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {

		wg.Add(1)
		go func() {
			start := time.Now()
			_, _, err := req.Get("https://www.baidu.com/").End()
			log.Println(err, time.Now().Sub(start).Seconds())
			wg.Done()
		}()
	}

	wg.Wait()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			start := time.Now()
			_, _, err := req.Get("https://www.baidu.com/").End()
			log.Println(err, time.Now().Sub(start).Seconds())
			wg.Done()
		}()
	}
	wg.Wait()

	for i := 0; i < 10; i++ {
		start := time.Now()
		_, _, err := req.Get("https://www.baidu.com/").End()
		log.Println(err, time.Now().Sub(start).Seconds())
	}
}
