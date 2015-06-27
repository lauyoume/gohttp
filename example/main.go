package main

import (
	"log"
	"time"

	"github.com/lauyoume/gohttp"
)

func main() {
	gohttp.SetOption(&gohttp.Option{
		Address: []string{"104.238.193.74", "104.238.193.75"},
	})

	req := gohttp.New()

	for i := 0; i < 10; i++ {
		start := time.Now()
		_, _, err := req.Get("https://www.baidu.com/").End()
		log.Println(err, time.Now().Sub(start).Seconds())
	}
}
