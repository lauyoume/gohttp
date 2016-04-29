package gohttp

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type Request *http.Request
type Response *http.Response

// HTTP methods we support
const (
	POST   = "POST"
	GET    = "GET"
	HEAD   = "HEAD"
	PUT    = "PUT"
	DELETE = "DELETE"
	PATCH  = "PATCH"
)

// A HttpAgent is a object storing all request data for client.
type HttpAgent struct {
	Url          string
	ProxyUrl     string
	Method       string
	Header       map[string]string
	TargetType   string
	ForceType    string
	Data         map[string]interface{}
	FormData     url.Values
	QueryData    url.Values
	Cookies      []*http.Cookie
	TlsConfig    *tls.Config
	MaxTimeout   time.Duration
	MaxRedirects int
	Client       *http.Client
	SingleClient bool
	Errors       []error
}

// Used to create a new HttpAgent object.
func New() *HttpAgent {
	s := &HttpAgent{
		TargetType:   "json",
		Data:         make(map[string]interface{}),
		Header:       make(map[string]string),
		FormData:     url.Values{},
		QueryData:    url.Values{},
		Cookies:      make([]*http.Cookie, 0),
		MaxRedirects: -1,
		Errors:       nil,
	}
	return s
}

func NewSingle() *HttpAgent {

	s := &HttpAgent{
		TargetType:   "json",
		Data:         make(map[string]interface{}),
		Header:       make(map[string]string),
		FormData:     url.Values{},
		QueryData:    url.Values{},
		Cookies:      make([]*http.Cookie, 0),
		MaxRedirects: -1,
		SingleClient: true,
		Errors:       nil,
	}
	return s
}

// Clear HttpAgent data for another new request.
func (s *HttpAgent) ClearAgent() {
	s.Url = ""
	s.Method = ""
	s.Header = make(map[string]string)
	s.Data = make(map[string]interface{})
	s.FormData = url.Values{}
	s.QueryData = url.Values{}
	s.ForceType = ""
	s.TargetType = "json"
	s.Cookies = make([]*http.Cookie, 0)
	s.Errors = nil
}

func (s *HttpAgent) Get(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = GET
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *HttpAgent) Post(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = POST
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *HttpAgent) Head(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = HEAD
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *HttpAgent) Put(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = PUT
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *HttpAgent) Delete(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = DELETE
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *HttpAgent) Patch(targetUrl string) *HttpAgent {
	s.ClearAgent()
	s.Method = PATCH
	s.Url = targetUrl
	s.Errors = nil
	return s
}

// Set is used for setting header fields.
// Example. To set `Accept` as `application/json`
//
//    gorequest.New().
//      Post("/gamelist").
//      Set("Accept", "application/json").
//      End()
func (s *HttpAgent) Set(param string, value string) *HttpAgent {
	s.Header[param] = value
	return s
}

// AddCookie adds a cookie to the request. The behavior is the same as AddCookie on Request from net/http
func (s *HttpAgent) AddCookie(c *http.Cookie) *HttpAgent {
	s.Cookies = append(s.Cookies, c)
	return s
}

var Types = map[string]string{
	"html":       "text/html",
	"json":       "application/json",
	"xml":        "application/xml",
	"urlencoded": "application/x-www-form-urlencoded",
	"form":       "application/x-www-form-urlencoded",
	"form-data":  "application/x-www-form-urlencoded",
	"text":       "text/plain",
}

// Type is a convenience function to specify the data type to send.
// For example, to send data as `application/x-www-form-urlencoded` :
//
//    gorequest.New().
//      Post("/recipe").
//      Type("form").
//      Send(`{ name: "egg benedict", category: "brunch" }`).
//      End()
//
// This will POST the body "name=egg benedict&category=brunch" to url /recipe
//
// GoRequest supports
//
//    "text/html" uses "html"
//    "application/json" uses "json"
//    "application/xml" uses "xml"
//    "application/x-www-form-urlencoded" uses "urlencoded", "form" or "form-data"
//
func (s *HttpAgent) Type(typeStr string) *HttpAgent {
	if _, ok := Types[typeStr]; ok {
		s.ForceType = typeStr
	} else {
		s.Errors = append(s.Errors, errors.New("Type func: incorrect type \""+typeStr+"\""))
	}
	return s
}

// Query function accepts either json string or strings which will form a query-string in url of GET method or body of POST method.
// For example, making "/search?query=bicycle&size=50x50&weight=20kg" using GET method:
//
//      gorequest.New().
//        Get("/search").
//        Query(`{ query: 'bicycle' }`).
//        Query(`{ size: '50x50' }`).
//        Query(`{ weight: '20kg' }`).
//        End()
//
// Or you can put multiple json values:
//
//      gorequest.New().
//        Get("/search").
//        Query(`{ query: 'bicycle', size: '50x50', weight: '20kg' }`).
//        End()
//
// Strings are also acceptable:
//
//      gorequest.New().
//        Get("/search").
//        Query("query=bicycle&size=50x50").
//        Query("weight=20kg").
//        End()
//
// Or even Mixed! :)
//
//      gorequest.New().
//        Get("/search").
//        Query("query=bicycle").
//        Query(`{ size: '50x50', weight:'20kg' }`).
//        End()
//
func (s *HttpAgent) Query(content interface{}) *HttpAgent {
	switch v := reflect.ValueOf(content); v.Kind() {
	case reflect.String:
		s.queryString(v.String())
	case reflect.Struct:
		s.queryStruct(v.Interface())
	default:
	}
	return s
}

func (s *HttpAgent) queryStruct(content interface{}) *HttpAgent {
	if marshalContent, err := json.Marshal(content); err != nil {
		s.Errors = append(s.Errors, err)
	} else {
		var val map[string]interface{}
		if err := json.Unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			for k, v := range val {
				k = strings.ToLower(k)
				s.QueryData.Add(k, v.(string))
			}
		}
	}
	return s
}

func (s *HttpAgent) queryString(content string) *HttpAgent {
	var val map[string]string
	if err := json.Unmarshal([]byte(content), &val); err == nil {
		for k, v := range val {
			s.QueryData.Add(k, v)
		}
	} else {
		if queryVal, err := url.ParseQuery(content); err == nil {
			for k, _ := range queryVal {
				s.QueryData.Add(k, queryVal.Get(k))
			}
		} else {
			s.Errors = append(s.Errors, err)
		}
		// TODO: need to check correct format of 'field=val&field=val&...'
	}
	return s
}

func (s *HttpAgent) Timeout(timeout time.Duration) *HttpAgent {
	s.MaxTimeout = timeout
	return s
}

// Set TLSClientConfig for underling Transport.
// One example is you can use it to disable security check (https):
//
// 			gorequest.New().TLSClientConfig(&tls.Config{ InsecureSkipVerify: true}).
// 				Get("https://disable-security-check.com").
// 				End()
//
func (s *HttpAgent) TLSClientConfig(config *tls.Config) *HttpAgent {
	s.TlsConfig = config
	return s
}

// Proxy function accepts a proxy url string to setup proxy url for any request.
// It provides a convenience way to setup proxy which have advantages over usual old ways.
// One example is you might try to set `http_proxy` environment. This means you are setting proxy up for all the requests.
// You will not be able to send different request with different proxy unless you change your `http_proxy` environment again.
// Another example is using Golang proxy setting. This is normal prefer way to do but too verbase compared to GoRequest's Proxy:
//
//      gorequest.New().Proxy("http://myproxy:9999").
//        Post("http://www.google.com").
//        End()
//
// To set no_proxy, just put empty string to Proxy func:
//
//      gorequest.New().Proxy("").
//        Post("http://www.google.com").
//        End()
//
func (s *HttpAgent) Proxy(proxyUrl string) *HttpAgent {
	s.ProxyUrl = proxyUrl
	return s
}

func (s *HttpAgent) MaxRedirect(redirect int) *HttpAgent {
	s.MaxRedirects = redirect
	return s
}

//func (s *HttpAgent) RedirectPolicy(policy func(req Request, via []Request) error) *HttpAgent {
//	s.Client.CheckRedirect = func(r *http.Request, v []*http.Request) error {
//		vv := make([]Request, len(v))
//		for i, r := range v {
//			vv[i] = Request(r)
//		}
//		return policy(Request(r), vv)
//	}
//	return s
//}

// Send function accepts either json string or query strings which is usually used to assign data to POST or PUT method.
// Without specifying any type, if you give Send with json data, you are doing requesting in json format:
//
//      gorequest.New().
//        Post("/search").
//        Send(`{ query: 'sushi' }`).
//        End()
//
// While if you use at least one of querystring, GoRequest understands and automatically set the Content-Type to `application/x-www-form-urlencoded`
//
//      gorequest.New().
//        Post("/search").
//        Send("query=tonkatsu").
//        End()
//
// So, if you want to strictly send json format, you need to use Type func to set it as `json` (Please see more details in Type function).
// You can also do multiple chain of Send:
//
//      gorequest.New().
//        Post("/search").
//        Send("query=bicycle&size=50x50").
//        Send(`{ wheel: '4'}`).
//        End()
//
// From v0.2.0, Send function provide another convenience way to work with Struct type. You can mix and match it with json and query string:
//
//      type BrowserVersionSupport struct {
//        Chrome string
//        Firefox string
//      }
//      ver := BrowserVersionSupport{ Chrome: "37.0.2041.6", Firefox: "30.0" }
//      gorequest.New().
//        Post("/update_version").
//        Send(ver).
//        Send(`{"Safari":"5.1.10"}`).
//        End()
//
func (s *HttpAgent) Send(content interface{}) *HttpAgent {
	// TODO: add normal text mode or other mode to Send func
	switch v := reflect.ValueOf(content); v.Kind() {
	case reflect.String:
		s.SendString(v.String())
	case reflect.Struct:
		s.sendStruct(v.Interface())
	default:
		// TODO: leave default for handling other types in the future such as number, byte, etc...
	}
	return s
}

// sendStruct (similar to SendString) returns HttpAgent's itself for any next chain and takes content interface{} as a parameter.
// Its duty is to transfrom interface{} (implicitly always a struct) into s.Data (map[string]interface{}) which later changes into appropriate format such as json, form, text, etc. in the End() func.
func (s *HttpAgent) sendStruct(content interface{}) *HttpAgent {
	if marshalContent, err := json.Marshal(content); err != nil {
		s.Errors = append(s.Errors, err)
	} else {
		var val map[string]interface{}
		if err := json.Unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			for k, v := range val {
				s.Data[k] = v
			}
		}
	}
	return s
}

// SendString returns HttpAgent's itself for any next chain and takes content string as a parameter.
// Its duty is to transform String into s.Data (map[string]interface{}) which later changes into appropriate format such as json, form, text, etc. in the End func.
// Send implicitly uses SendString and you should use Send instead of this.
func (s *HttpAgent) SendString(content string) *HttpAgent {
	if s.ForceType == "text" || s.ForceType == "xml" {
		s.Data["text"] = content
		//s.TargetType = s.ForceType
		return s
	}
	var val map[string]interface{}
	// check if it is json format
	if err := json.Unmarshal([]byte(content), &val); err == nil {
		for k, v := range val {
			s.Data[k] = v
		}
	} else if formVal, err := url.ParseQuery(content); err == nil {
		for k, _ := range formVal {
			// make it array if already have key
			if val, ok := s.Data[k]; ok {
				var strArray []string
				strArray = append(strArray, formVal.Get(k))
				// check if previous data is one string or array
				switch oldValue := val.(type) {
				case []string:
					strArray = append(strArray, oldValue...)
				case string:
					strArray = append(strArray, oldValue)
				}
				s.Data[k] = strArray
			} else {
				// make it just string if does not already have same key
				s.Data[k] = formVal.Get(k)
			}
		}
		s.TargetType = "form"
	} else {
		// need to add text mode or other format body request to this func
	}
	return s
}

func changeMapToURLValues(data map[string]interface{}) url.Values {
	var newUrlValues = url.Values{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			newUrlValues.Add(k, val)
		case []string:
			for _, element := range val {
				newUrlValues.Add(k, element)
			}
		}
	}
	return newUrlValues
}

// End is the most important function that you need to call when ending the chain. The request won't proceed without calling it.
// End function returns Response which matchs the structure of Response type in Golang's http package (but without Body data). The body data itself returns as a string in a 2nd return value.
// Lastly but worht noticing, error array (NOTE: not just single error value) is returned as a 3rd value and nil otherwise.
//
// For example:
//
//    resp, body, errs := gorequest.New().Get("http://www.google.com").End()
//    if( errs != nil){
//      fmt.Println(errs)
//    }
//    fmt.Println(resp, body)
//
// Moreover, End function also supports callback which you can put as a parameter.
// This extends the flexibility and makes GoRequest fun and clean! You can use GoRequest in whatever style you love!
//
// For example:
//
//    func printBody(resp gorequest.Response, body string, errs []error){
//      fmt.Println(resp.Status)
//    }
//    gorequest.New().Get("http://www..google.com").End(printBody)
//
func (s *HttpAgent) End(callback ...func(response Response, errs []error)) (Response, []error) {
	var (
		req            *http.Request
		err            error
		resp           Response
		redirectFailed bool
		client         *http.Client
	)
	// check whether there is an error. if yes, return all errors
	if len(s.Errors) != 0 {
		return nil, s.Errors
	}

	if s.Client != nil {
		client = s.Client
	} else {
		client, err = GetHttpClient(s.Url, s.ProxyUrl)
		if err != nil {
			s.Errors = append(s.Errors, err)
			return nil, s.Errors
		}
		if s.SingleClient {
			s.Client = client
		}
	}
	transport, _ := client.Transport.(*http.Transport)

	// check if there is forced type
	switch s.ForceType {
	case "json", "form", "text", "xml":
		s.TargetType = s.ForceType
	}

	switch s.Method {
	case POST, PUT, PATCH:
		if s.TargetType == "json" {
			contentJson, _ := json.Marshal(s.Data)
			contentReader := bytes.NewReader(contentJson)
			req, err = http.NewRequest(s.Method, s.Url, contentReader)
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		} else if s.TargetType == "form" {
			formData := changeMapToURLValues(s.Data)
			req, err = http.NewRequest(s.Method, s.Url, strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else if s.TargetType == "text" {
			formdata := s.Data["text"].(string)
			req, err = http.NewRequest(s.Method, s.Url, strings.NewReader(formdata))
			req.Header.Set("Content-Type", "text/plain")
		} else if s.TargetType == "xml" {
			formdata := s.Data["text"].(string)
			req, err = http.NewRequest(s.Method, s.Url, strings.NewReader(formdata))
			req.Header.Set("Content-Type", "text/xml")
		}
	case GET, HEAD, DELETE:
		req, err = http.NewRequest(s.Method, s.Url, nil)
	}

	if _, ok := s.Header["User-Agent"]; !ok {
		s.Header["User-Agent"] = defaultOption.Agent
	}
	for k, v := range s.Header {
		req.Header.Set(k, v)
	}
	// Add all querystring from Query func
	if len(s.QueryData) > 0 {
		q := req.URL.Query()
		for k, v := range s.QueryData {
			for _, vv := range v {
				q.Add(k, vv)
			}
		}
		req.URL.RawQuery = q.Encode()
	}

	// Add cookies
	for _, cookie := range s.Cookies {
		req.AddCookie(cookie)
	}

	if s.TlsConfig != nil {
		transport.TLSClientConfig = s.TlsConfig
	} else if transport.TLSClientConfig != nil {
		transport.TLSClientConfig.InsecureSkipVerify = false
		//client.Transport.TLSClientConfig = nil
	}

	if s.MaxRedirects == -1 {
		s.MaxRedirects = defaultOption.MaxRedirects
	}
	if s.MaxRedirects >= 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > s.MaxRedirects {
				redirectFailed = true
				return errors.New("Error redirecting. MaxRedirects reached")
			}

			//By default Golang will not redirect request headers
			// https://code.google.com/p/go/issues/detail?id=4800&q=request%20header
			for key, val := range via[0].Header {
				req.Header[key] = val
			}
			return nil
		}
	}

	//timeout := false
	//var timer *time.Timer
	//if s.MaxTimeout > 0 {
	//	//timer = time.AfterFunc(s.MaxTimeout, func() {
	//	//	transport.CancelRequest(req)
	//	//	timeout = true
	//	//})
	//}
	client.Timeout = s.MaxTimeout
	// Send request
	resp, err = client.Do(req)
	//if timer != nil {
	//	timer.Stop()
	//}

	if err != nil {
		s.Errors = append(s.Errors, err)
		return resp, s.Errors
	}
	// deep copy response to give it to both return and callback func
	respCallback := *resp
	if len(callback) != 0 {
		callback[0](&respCallback, s.Errors)
	}
	return resp, nil
}