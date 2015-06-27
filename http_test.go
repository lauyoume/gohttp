package gohttp

import (
	"log"
	"testing"
	"time"
)

func TestGoHttp(t *testing.T) {

	SetOption(&Option{
		Delay: time.Second * 10,
	})
	req := New()

	resp, _, _ := req.Get("http://www.baidu.com").End()
	log.Println(resp)
	resp2, _, _ := req.Get("https://www.baidu.com/s?wd=%E5%A5%B3%E5%8C%AA%E9%A6%96&rsv_spt=1&issp=1&f=3&rsv_bp=0&rsv_idx=2&ie=utf-8&tn=baiduhome_pg&rsv_enter=1&rsv_sug3=36&rsv_sug1=22&oq=nvuf&rsv_sug2=1&rsp=0&inputT=3748547&rsv_sug4=3749526").End()
	log.Println(resp2)
}
