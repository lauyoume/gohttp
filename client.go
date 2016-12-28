package gohttp

import (
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
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
	MaxIdleConns   int
}

type clientResource struct {
	Transport http.RoundTripper
	Jar       http.CookieJar
}

type useInfo struct {
	Index    int
	LastTime time.Time
}

var defaultOption = &Option{
	ConnectTimeout: 30000 * time.Millisecond,
	Agent:          "gohttp v1.0",
	Address:        make([]string, 0),
	MaxRedirects:   -1,
	MaxIdleConns:   0,
}

//ip使用情况
var useLock sync.RWMutex
var useMap map[string]*useInfo = make(map[string]*useInfo)
var clientMap map[string]*clientResource
var clientLock sync.RWMutex

var defaultDialer = &net.Dialer{Timeout: defaultOption.ConnectTimeout}
var defaultTransport = &http.Transport{Dial: defaultDialer.Dial, Proxy: http.ProxyFromEnvironment}
var defaultCookiejar = MakeCookiejar()

var proxyTransport *http.Transport

var hostDelay = make(map[string]time.Duration)
var hostDelayLock sync.RWMutex

func MakeCookiejar() http.CookieJar {
	cookiejarOptions := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&cookiejarOptions)

	return jar
}

func MakeClient(transport http.RoundTripper, jar http.CookieJar) *http.Client {
	return &http.Client{Jar: jar, Transport: transport, Timeout: 60 * time.Second}
}

func MakeTransport(ip string) *http.Transport {
	addr, _ := net.ResolveTCPAddr("tcp", ip+":0")
	dialer := &net.Dialer{
		Timeout:   defaultOption.ConnectTimeout,
		LocalAddr: addr,
	}
	transport := &http.Transport{
		Dial:                dialer.Dial,
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConnsPerHost: defaultOption.MaxIdleConns,
	}

	if defaultOption.MaxIdleConns <= 0 {
		transport.DisableKeepAlives = true
	}

	return transport
}

func SetHostDelay(host string, delay time.Duration) {
	defer hostDelayLock.Unlock()
	hostDelayLock.Lock()
	if d, ok := hostDelay[host]; ok && delay > d {
		hostDelay[host] = delay
		return
	}
	hostDelay[host] = delay
}

func GetHostDelay(host string) time.Duration {
	defer hostDelayLock.RUnlock()
	hostDelayLock.RLock()

	if d, ok := hostDelay[host]; ok {
		return d
	}

	return defaultOption.Delay
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
		clientMap = make(map[string]*clientResource)
	}

	if option.MaxRedirects > 0 {
		defaultOption.MaxRedirects = option.MaxRedirects
	}

	if option.MaxIdleConns > 0 {
		defaultOption.MaxIdleConns = option.MaxIdleConns
		defaultTransport.MaxIdleConnsPerHost = option.MaxIdleConns
	}
}

func ResetCookie(urlstr string) error {
	uri, err := url.Parse(urlstr)
	if err != nil {
		return err
	}
	clientLock.Lock()

	cookies := defaultCookiejar.Cookies(uri)
	for _, c := range cookies {
		c.Expires = time.Now().Add(-1 * time.Hour)
	}
	defaultCookiejar.SetCookies(uri, cookies)

	for _, client := range clientMap {
		cookies := client.Jar.Cookies(uri)
		for _, c := range cookies {
			c.Expires = time.Now().Add(-1 * time.Hour)
		}
		client.Jar.SetCookies(uri, cookies)
	}
	clientLock.Unlock()
	return nil
}

func GetHttpClient(urlStr string, proxy string, usejar bool) (*http.Client, error) {

	var clientres *clientResource
	if proxy != "" {
		proxyuri, err := url.Parse(proxy)
		if err != nil {
			return nil, err
		}
		if proxyTransport == nil {
			proxyTransport = &http.Transport{Dial: defaultDialer.Dial, Proxy: http.ProxyURL(proxyuri)}
		} else {
			proxyTransport.Proxy = http.ProxyURL(proxyuri)
		}
		clientres = &clientResource{proxyTransport, defaultCookiejar}
	} else {

		uri, err := url.Parse(urlStr)
		if err != nil {
			return nil, err
		}
		delay := time.Duration(0)

		//并发取的时候锁定
		useLock.Lock()
		use, ok := useMap[uri.Host]
		need_delay := GetHostDelay(uri.Host)
		if ok {
			//need_delay
			lastIndex := use.Index
			if len(defaultOption.Address) != 0 {
				use.Index = (use.Index + 1) % len(defaultOption.Address)
			}

			//使用同一个IP，则需要延迟
			if lastIndex == use.Index && need_delay > 0 {
				sub := time.Now().Sub(use.LastTime)
				if sub < need_delay {
					delay = need_delay - sub
				}
			}
			use.LastTime = time.Now().Add(delay)
		} else {
			use = &useInfo{
				Index:    0,
				LastTime: time.Now(),
			}
		}
		useMap[uri.Host] = use
		useLock.Unlock()

		if delay > 0 {
			time.Sleep(delay)
		}

		if len(defaultOption.Address) == 0 {
			clientres = &clientResource{defaultTransport, defaultCookiejar}
		} else {
			//
			//加锁并发
			ip := defaultOption.Address[use.Index]
			clientLock.Lock()
			if v, ok := clientMap[ip]; ok {
				clientres = v
			} else {
				clientres = &clientResource{MakeTransport(ip), MakeCookiejar()}
				clientMap[ip] = clientres
			}
			clientLock.Unlock()
		}

	}

	if usejar {
		return MakeClient(clientres.Transport, clientres.Jar), nil
	}
	return MakeClient(clientres.Transport, MakeCookiejar()), nil
}

func GetDefaultDialer() *net.Dialer {
	return defaultDialer
}

func GetDefaultTransport() *http.Transport {
	return defaultTransport
}

func GetDefaultClient() *http.Client {
	return MakeClient(defaultTransport, defaultCookiejar)
}
