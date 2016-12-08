package gohttp

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
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

func TestQuery(t *testing.T) {

	SetOption(&Option{
		Delay: time.Second * 10,
	})
	req := New()

	resp, _ := req.Get("http://www.baidu.com/s").Query(map[string]interface{}{
		"rn": 50,
		"pn": 100,
		"wd": "女神",
	}).End()

	defer resp.Body.Close()

	fmt.Println(resp.Request.URL)
	doc, _ := goquery.NewDocumentFromReader(resp.Body)

	fmt.Println(doc.Find("title").Text())
	fmt.Println(doc.Find(".result h3 a").Text())
	fmt.Println(doc.Find("#page").Html())
}
