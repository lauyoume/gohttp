package gohttp

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

//type Request *http.Request
//type Response *http.Response

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
	FileData     []File
	Cookies      []*http.Cookie
	TlsConfig    *tls.Config
	MaxTimeout   time.Duration
	MaxRedirects int
	Client       *http.Client
	SingleClient bool
	Usejar       bool
	Errors       []error
	DataAll      interface{}
	Getter       ClientGetter
}

// Used to create a new HttpAgent object.
func New() *HttpAgent {
	s := &HttpAgent{
		TargetType:   "json",
		Data:         make(map[string]interface{}),
		Header:       make(map[string]string),
		FormData:     url.Values{},
		QueryData:    url.Values{},
		FileData:     make([]File, 0),
		Cookies:      make([]*http.Cookie, 0),
		MaxRedirects: -1,
		Errors:       nil,
		Usejar:       true,
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
		FileData:     make([]File, 0),
		Cookies:      make([]*http.Cookie, 0),
		MaxRedirects: -1,
		SingleClient: true,
		Errors:       nil,
		Usejar:       true,
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
	s.FileData = make([]File, 0)
	s.ForceType = ""
	s.TargetType = "json"
	s.Cookies = make([]*http.Cookie, 0)
	s.Errors = nil
	s.DataAll = nil
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
//    gohttp.New().
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
	"multipart":  "multipart/form-data",
	"stream":     "application/octet-stream",
}

// Type is a convenience function to specify the data type to send.
// For example, to send data as `application/x-www-form-urlencoded` :
//
//    gohttp.New().
//      Post("/recipe").
//      Type("form").
//      Send(`{ name: "egg benedict", category: "brunch" }`).
//      End()
//
// This will POST the body "name=egg benedict&category=brunch" to url /recipe
//
// gohttp supports
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
//      gohttp.New().
//        Get("/search").
//        Query(`{ query: 'bicycle' }`).
//        Query(`{ size: '50x50' }`).
//        Query(`{ weight: '20kg' }`).
//        End()
//
// Or you can put multiple json values:
//
//      gohttp.New().
//        Get("/search").
//        Query(`{ query: 'bicycle', size: '50x50', weight: '20kg' }`).
//        End()
//
// Strings are also acceptable:
//
//      gohttp.New().
//        Get("/search").
//        Query("query=bicycle&size=50x50").
//        Query("weight=20kg").
//        End()
//
// Or even Mixed! :)
//
//      gohttp.New().
//        Get("/search").
//        Query("query=bicycle").
//        Query(`{ size: '50x50', weight:'20kg' }`).
//        End()
//
func (s *HttpAgent) Query(content interface{}) *HttpAgent {
	switch v := reflect.ValueOf(content); v.Kind() {
	case reflect.String:
		s.queryString(v.String())
	case reflect.Struct, reflect.Map:
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
		if err := json_unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			newdata := changeMapToURLValues(val)
			for k, v := range newdata {
				for _, v1 := range v {
					s.QueryData.Add(k, v1)
				}
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

// As Go conventions accepts ; as a synonym for &. (https://github.com/golang/go/issues/2210)
// Thus, Query won't accept ; in a querystring if we provide something like fields=f1;f2;f3
// This Param is then created as an alternative method to solve this.
func (s *HttpAgent) Param(key string, value string) *HttpAgent {
	s.QueryData.Add(key, value)
	return s
}

func (s *HttpAgent) Timeout(timeout time.Duration) *HttpAgent {
	s.MaxTimeout = timeout
	return s
}

// Set TLSClientConfig for underling Transport.
// One example is you can use it to disable security check (https):
//
// 			gohttp.New().TLSClientConfig(&tls.Config{ InsecureSkipVerify: true}).
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
// Another example is using Golang proxy setting. This is normal prefer way to do but too verbase compared to gohttp's Proxy:
//
//      gohttp.New().Proxy("http://myproxy:9999").
//        Post("http://www.google.com").
//        End()
//
// To set no_proxy, just put empty string to Proxy func:
//
//      gohttp.New().Proxy("").
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
//      gohttp.New().
//        Post("/search").
//        Send(`{ query: 'sushi' }`).
//        End()
//
// While if you use at least one of querystring, gohttp understands and automatically set the Content-Type to `application/x-www-form-urlencoded`
//
//      gohttp.New().
//        Post("/search").
//        Send("query=tonkatsu").
//        End()
//
// So, if you want to strictly send json format, you need to use Type func to set it as `json` (Please see more details in Type function).
// You can also do multiple chain of Send:
//
//      gohttp.New().
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
//      gohttp.New().
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
	case reflect.Array, reflect.Slice:
		s.sendArray(v.Interface())
	case reflect.Struct, reflect.Map:
		s.sendStruct(v.Interface())
	default:
		// TODO: leave default for handling other types in the future such as number, byte, etc...
	}
	return s
}

func (s *HttpAgent) sendArray(content interface{}) *HttpAgent {
	if marshalContent, err := json.Marshal(content); err != nil {
		s.Errors = append(s.Errors, err)
	} else {
		var val []interface{}
		if err := json_unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			s.DataAll = val
		}
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
		if err := json_unmarshal(marshalContent, &val); err != nil {
			s.Errors = append(s.Errors, err)
		} else {
			for k, v := range val {
				s.Data[k] = v
			}
		}
	}
	return s
}

func (s *HttpAgent) SendBytes(data []byte) *HttpAgent {
	if s.ForceType == "stream" {
		s.Data["stream"] = data
		return s
	}

	return s.SendString(string(data))
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
	var valslice []interface{}
	// check if it is json format
	if err := json_unmarshal([]byte(content), &val); err == nil {
		for k, v := range val {
			s.Data[k] = v
		}
	} else if err := json_unmarshal([]byte(content), &valslice); err == nil {
		s.DataAll = valslice
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

type File struct {
	Filename  string
	Fieldname string
	Reader    io.Reader
	Len       int64
}

// SendFile function works only with type "multipart". The function accepts one mandatory and up to two optional arguments. The mandatory (first) argument is the file.
// The function accepts a path to a file as string:
//
//      gorequest.New().
//        Post("http://example.com").
//        Type("multipart").
//        SendFile("./example_file.ext").
//        End()
//
// File can also be a []byte slice of a already file read by eg. ioutil.ReadFile:
//
//      b, _ := ioutil.ReadFile("./example_file.ext")
//      gorequest.New().
//        Post("http://example.com").
//        Type("multipart").
//        SendFile(b).
//        End()
//
// Furthermore file can also be a os.File:
//
//      f, _ := os.Open("./example_file.ext")
//      gorequest.New().
//        Post("http://example.com").
//        Type("multipart").
//        SendFile(f).
//        End()
//
// The first optional argument (second argument overall) is the filename, which will be automatically determined when file is a string (path) or a os.File.
// When file is a []byte slice, filename defaults to "filename". In all cases the automatically determined filename can be overwritten:
//
//      b, _ := ioutil.ReadFile("./example_file.ext")
//      gorequest.New().
//        Post("http://example.com").
//        Type("multipart").
//        SendFile(b, "my_custom_filename").
//        End()
//
// The second optional argument (third argument overall) is the fieldname in the multipart/form-data request. It defaults to fileNUMBER (eg. file1), where number is ascending and starts counting at 1.
// So if you send multiple files, the fieldnames will be file1, file2, ... unless it is overwritten. If fieldname is set to "file" it will be automatically set to fileNUMBER, where number is the greatest exsiting number+1.
//
//      b, _ := ioutil.ReadFile("./example_file.ext")
//      gorequest.New().
//        Post("http://example.com").
//        Type("multipart").
//        SendFile(b, "", "my_custom_fieldname"). // filename left blank, will become "example_file.ext"
//        End()
//
// 大文件建议传os.File进来
func (s *HttpAgent) SendFile(file interface{}, args ...string) *HttpAgent {

	filename := ""
	fieldname := "file"

	if len(args) >= 1 && len(args[0]) > 0 {
		filename = strings.TrimSpace(args[0])
	}
	if len(args) >= 2 && len(args[1]) > 0 {
		fieldname = strings.TrimSpace(args[1])
	}

	//if fieldname == "file" || fieldname == "" {
	//	fieldname = "file" + strconv.Itoa(len(s.FileData)+1)
	//}

	switch v := file.(type) {
	case string:
		pathToFile, err := filepath.Abs(v)
		if err != nil {
			s.Errors = append(s.Errors, err)
			return s
		}
		if filename == "" {
			filename = filepath.Base(pathToFile)
		}
		data, err := ioutil.ReadFile(v)
		if err != nil {
			s.Errors = append(s.Errors, err)
			return s
		}
		s.FileData = append(s.FileData, File{
			Filename:  filename,
			Fieldname: fieldname,
			Reader:    bytes.NewReader(data),
			Len:       int64(len(v)),
		})
	case []byte:
		if filename == "" {
			filename = "filename"
		}
		f := File{
			Filename:  filename,
			Fieldname: fieldname,
			Reader:    bytes.NewReader(v),
			Len:       int64(len(v)),
		}
		s.FileData = append(s.FileData, f)
	case *os.File:
		osfile := v
		if filename == "" {
			filename = filepath.Base(osfile.Name())
		}
		stat, _ := osfile.Stat()
		s.FileData = append(s.FileData, File{
			Filename:  filename,
			Fieldname: fieldname,
			Len:       stat.Size(),
			Reader:    osfile,
		})
	default:
		s.Errors = append(s.Errors, errors.New("SendFile currently only supports either a string (path/to/file), a bytes (file content itself), or a os.File!"))
	}

	return s
}

func changeMapToURLValues(data map[string]interface{}) url.Values {
	var newUrlValues = url.Values{}
	for k, v := range data {
		switch val := v.(type) {
		case bool:
			if val {
				newUrlValues.Add(k, "1")
			} else {
				newUrlValues.Add(k, "0")
			}
		case json.Number:
			newUrlValues.Add(k, string(val))
		case int, int8, int16, int32, int64, float64, float32:
			newUrlValues.Add(k, fmt.Sprintf("%v", val))
		case uint, uint8, uint16, uint32, uint64:
			newUrlValues.Add(k, fmt.Sprintf("%v", val))
		case string:
			newUrlValues.Add(k, val)
		case []int, []int64, []float64, []interface{}:
			v := reflect.ValueOf(val)
			for i := 0; i < v.Len(); i++ {
				newUrlValues.Add(fmt.Sprintf("%s[]", k), fmt.Sprintf("%v", v.Index(i).Interface()))
			}
		case []string:
			for _, element := range val {
				newUrlValues.Add(fmt.Sprintf("%s[]", k), element)
			}
		default:
			body, _ := json.Marshal(val)
			newUrlValues.Add(k, string(body))
		}
	}

	return newUrlValues
}

func (s *HttpAgent) Jar(use bool) *HttpAgent {
	s.Usejar = use
	return s
}

// End is the most important function that you need to call when ending the chain. The request won't proceed without calling it.
// End function returns Response which matchs the structure of Response type in Golang's http package (but without Body data). The body data itself returns as a string in a 2nd return value.
// Lastly but worht noticing, error array (NOTE: not just single error value) is returned as a 3rd value and nil otherwise.
//
// For example:
//
//    resp, body, errs := gohttp.New().Get("http://www.google.com").End()
//    if( errs != nil){
//      fmt.Println(errs)
//    }
//    fmt.Println(resp, body)
//
// Moreover, End function also supports callback which you can put as a parameter.
// This extends the flexibility and makes gohttp fun and clean! You can use gohttp in whatever style you love!
//
// For example:
//
//    func printBody(resp gohttp.Response, body string, errs []error){
//      fmt.Println(resp.Status)
//    }
//    gohttp.New().Get("http://www..google.com").End(printBody)
//
func (s *HttpAgent) End(callback ...func(response *http.Response, errs []error)) (*http.Response, []error) {
	var (
		req    *http.Request
		err    error
		resp   *http.Response
		client *http.Client
	)
	// check whether there is an error. if yes, return all errors
	if len(s.Errors) != 0 {
		return nil, s.Errors
	}

	if s.Client != nil {
		client = s.Client
	} else {
		getter := GetDefaultGetter()
		if s.Getter != nil {
			getter = s.Getter
		}

		client, err = getter.GetHttpClient(s.Url, s.ProxyUrl, s.Usejar)
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
	case "json", "form", "text", "xml", "multipart", "stream":
		s.TargetType = s.ForceType
	}

	switch s.Method {
	case POST, PUT, PATCH:
		if s.TargetType == "json" {
			var contentJson []byte
			if s.DataAll != nil {
				contentJson, _ = json.Marshal(s.DataAll)
			} else {
				contentJson, _ = json.Marshal(s.Data)
			}
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
		} else if s.TargetType == "stream" {
			body := s.Data["stream"].([]byte)
			req, err = http.NewRequest(s.Method, s.Url, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/octet-stream")
		} else if s.TargetType == "multipart" {

			mw := NewMultiPartStreamer()

			if len(s.Data) != 0 {
				formData := changeMapToURLValues(s.Data)
				mw.WriteFields(formData)
			}

			if len(s.FileData) > 0 {
				// 暂时只支持单个文件
				for _, file := range s.FileData {
					mw.WriteReader(file.Fieldname, file.Filename, file.Len, file.Reader)
				}
			}

			req, err = http.NewRequest(s.Method, s.Url, nil)
			mw.SetupRequest(req)
			// req.Header.Set("Content-Type", mw.FormDataContentType())
		}
	case GET, HEAD, DELETE:
		req, err = http.NewRequest(s.Method, s.Url, nil)
	}

	if _, ok := s.Header["User-Agent"]; !ok {
		s.Header["User-Agent"] = defaultOption.Agent
	}

	if host, ok := s.Header["Host"]; ok {
		req.Host = host
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
	} else if transport != nil && transport.TLSClientConfig != nil {
		transport.TLSClientConfig.InsecureSkipVerify = false
		//client.Transport.TLSClientConfig = nil
	}

	if s.MaxRedirects == -1 {
		s.MaxRedirects = defaultOption.MaxRedirects
	}
	if s.MaxRedirects >= 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) > s.MaxRedirects {
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

func (s *HttpAgent) Bytes(status ...int) ([]byte, int, error) {
	if s.Url == "" || s.Method == "" {
		return nil, http.StatusBadRequest, errors.New("req error, need set url and method")
	}

	resp, errs := s.End()
	if errs != nil {
		return nil, http.StatusBadRequest, errs[0]
	}
	defer resp.Body.Close()
	if status != nil {
		found := false
		for _, val := range status {
			if resp.StatusCode == val {
				found = true
				break
			}
		}
		if !found {
			io.Copy(ioutil.Discard, resp.Body)
			return nil, resp.StatusCode, errors.New(fmt.Sprintf("status not match we want!, statuscode = %d", resp.StatusCode))
		}
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}
		body, err := ioutil.ReadAll(reader)
		return body, resp.StatusCode, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (s *HttpAgent) String(status ...int) (string, int, error) {
	body, code, err := s.Bytes(status...)
	if err != nil {
		return "", code, err
	}

	return string(body), code, err
}

func (s *HttpAgent) ToJSON(v interface{}, status ...int) (int, error) {
	body, code, err := s.Bytes(status...)
	if err != nil {
		return code, err
	}

	err = json_unmarshal(body, &v)
	return code, err
}

func (s *HttpAgent) ToXML(v interface{}, status ...int) (int, error) {
	body, code, err := s.Bytes(status...)
	if err != nil {
		return code, err
	}

	err = xml.Unmarshal(body, &v)
	return code, err
}

func json_unmarshal(body []byte, v interface{}) error {
	d := json.NewDecoder(bytes.NewBuffer(body))
	d.UseNumber()

	return d.Decode(v)
}
