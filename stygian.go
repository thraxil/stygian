package main

import (
	"bytes"
	"encoding/json"
	_ "fmt"
	zmq "github.com/alecthomas/gozmq"
	"github.com/elazarl/goproxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var PUB_SOCKET = "tcp://*:6666"
var pubsocket zmq.Socket

type message struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

type BodyHandler struct {
	R    io.ReadCloser
	W    *bytes.Buffer
	Resp *http.Response
}

func (c *BodyHandler) Read(b []byte) (n int, err error) {
	n, err = c.R.Read(b)
	if n > 0 {
		if n, err := c.W.Write(b[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (c *BodyHandler) Close() error {
	contentType := c.Resp.Header.Get("Content-Type")
	content := c.W.String()
	m := message{c.Resp.Request.URL.String(), contentType, content}
	b, _ := json.Marshal(m)
	pubsocket.Send([]byte(b), 0)
	return c.R.Close()
}

func filter(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil {
		return resp
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/") {
		if strings.HasPrefix(contentType, "text/css") ||
			strings.HasPrefix(contentType, "text/javascript") ||
			strings.HasPrefix(contentType, "text/json") {
			return resp
		}
		// full response with a body. save it, index it, etc.

		length := resp.ContentLength
		if length <= 0 {
			length = 1024
		}
		buf := bytes.NewBuffer(make([]byte, 0, length))
		resp.Body = &BodyHandler{resp.Body, buf, resp}
	}

	return resp
}

func UrlSuffixMatches(suffixes ...string) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(req.URL.Path, suffix) {
				return true
			}
		}
		return false
	}
}

func StatusIs(status int) goproxy.RespCondition {
	return goproxy.RespConditionFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
		return resp.StatusCode == status
	})
}

func UrlMatchesAny(res ...*regexp.Regexp) goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		for _, re := range res {
			result := re.MatchString(req.URL.Path) ||
				re.MatchString(req.URL.Host+req.URL.Path)
			if result {
				// short-circuit
				return result
			}
		}
		return false
	}
}

func main() {
	var host_blacklist = []*regexp.Regexp{}
	var full_blacklist = []*regexp.Regexp{}
	var path_suffix_blacklist = []string{}

	context, _ := zmq.NewContext()
	pubsocket, _ = context.NewSocket(zmq.PUB)
	defer context.Close()
	defer pubsocket.Close()
	pubsocket.Bind(PUB_SOCKET)


	content, err := ioutil.ReadFile("domain_blacklist.txt")
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" {
				host_blacklist = append(host_blacklist, regexp.MustCompile(line))
			}
		}
	}

	content, err = ioutil.ReadFile("full_blacklist.txt")
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" {
				full_blacklist = append(full_blacklist, regexp.MustCompile(line))
			}
		}
	}

	content, err = ioutil.ReadFile("suffix_blacklist.txt")
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" {
				path_suffix_blacklist = append(path_suffix_blacklist, line)
			}
		}
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.OnResponse(
		StatusIs(200),
		goproxy.Not(UrlSuffixMatches(path_suffix_blacklist...)),
		goproxy.Not(goproxy.ReqHostMatches(host_blacklist...)),
		goproxy.Not(UrlMatchesAny(full_blacklist...)),
		).DoFunc(filter)
	log.Fatal(http.ListenAndServe("localhost:8080", proxy))
}
