package gohttp

import (
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Option struct {
	Address        []string
	ConnectTimeout time.Duration
	Timeout        time.Duration
	Agent          string
	Delay          time.Duration
	MaxRedirects   int
}

type useInfo struct {
	Index    int
	LastTime time.Time
}

var defaultOption = &Option{
	ConnectTimeout: 3000 * time.Millisecond,
	Agent:          "gohttp v1.0",
	Address:        make([]string, 0),
}

//ip使用情况
var useMap map[string]*useInfo = make(map[string]*useInfo)
var clientMap map[string]*http.Client

var defaultDialer = &net.Dialer{Timeout: defaultOption.ConnectTimeout}
var defaultTransport = &http.Transport{Dial: defaultDialer.Dial, Proxy: http.ProxyFromEnvironment}
var defaultClient = makeClient(defaultTransport)

var proxyClient *http.Client
var proxyTransport *http.Transport

func makeClient(transport *http.Transport) *http.Client {
	cookiejarOptions := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&cookiejarOptions)
	return &http.Client{Jar: jar, Transport: transport}
}

func makeTransport(ip string) *http.Transport {
	addr, _ := net.ResolveTCPAddr("tcp", ip+":0")
	dialer := &net.Dialer{
		Timeout:   defaultOption.ConnectTimeout,
		LocalAddr: addr,
	}
	return &http.Transport{
		Dial:  dialer.Dial,
		Proxy: http.ProxyFromEnvironment,
	}

}

func SetOption(option *Option) {
	if option.Agent != "" {
		defaultOption.Agent = option.Agent
	}

	if option.ConnectTimeout > 0 {
		defaultOption.ConnectTimeout = option.ConnectTimeout
	}

	if option.Delay > 0 {
		defaultOption.Delay = option.Delay
	}

	if option.Address != nil && len(option.Address) > 0 {
		defaultOption.Address = append(defaultOption.Address, option.Address...)
		clientMap = make(map[string]*http.Client)
	}

	if option.MaxRedirects > 0 {
		defaultOption.MaxRedirects = option.MaxRedirects
	}
}

func GetHttpClient(urlStr string, proxy string) (*http.Client, error) {

	var client *http.Client
	if proxy != "" {
		proxyuri, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		if proxyTransport == nil {
			proxyTransport = &http.Transport{Dial: defaultDialer.Dial, Proxy: http.ProxyURL(proxyuri)}
			proxyClient = makeClient(proxyTransport)
		} else {
			proxyTransport.Proxy = http.ProxyURL(proxyuri)
		}
		client = proxyClient
	} else {

		uri, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		use, ok := useMap[uri.Host]
		if ok {
			//need_delay
			lastIndex := use.Index
			if len(defaultOption.Address) != 0 {
				use.Index = (use.Index + 1) % len(defaultOption.Address)
			}

			//使用同一个IP，则需要延迟
			if lastIndex == use.Index && defaultOption.Delay > 0 {
				sub := time.Now().Sub(use.LastTime)
				if sub < defaultOption.Delay {
					time.Sleep(defaultOption.Delay - sub)
				}
			}
			use.LastTime = time.Now()
		} else {
			use = &useInfo{
				Index:    0,
				LastTime: time.Now(),
			}
		}
		useMap[uri.Host] = use

		if len(defaultOption.Address) == 0 {
			client = defaultClient
		} else {
			//
			ip := defaultOption.Address[use.Index]
			if v, ok := clientMap[ip]; ok {
				client = v
			} else {
				client = makeClient(makeTransport(ip))
				clientMap[ip] = client
			}
		}

	}

	return client, nil
}
