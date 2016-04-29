package gohttp

import (
	"io/ioutil"
	"log"
	"testing"
	"time"
)

func TestGoHttp(t *testing.T) {

	SetOption(&Option{
		Delay: time.Second * 10,
	})
	req := New()

	resp, _ := req.Get("http://www.baidu.com").End()
	log.Println(resp)
	resp2, errs := New().Get("http://www.im-reg.com/fgcl/").MaxRedirect(1).Set("User-Agent", "baiduspider").End()
	log.Println(resp2, errs)

	body, _ := ioutil.ReadAll(resp2.Body)
	log.Println(string(body))
}
