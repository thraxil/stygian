package main

import (
	"bytes"
	"encoding/json"
	"flag"
	_ "fmt"
	"github.com/elazarl/goproxy"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var SUBMIT_URL string

type message struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

type ConfigData struct {
	SubmitURL           string `json:"submit_url"`
	DomainBlacklistFile string `json:"domain_blacklist_file"`
	FullBlacklistFile   string `json:"full_blacklist_file"`
	SuffixBlacklistFile string `json:"suffix_blacklist_file"`
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
	go submit(c.Resp.Request.URL.String(),
		contentType,
		content)
	return c.R.Close()
}

func submit(url_visited, content_type, body string) {
	http.PostForm(SUBMIT_URL,
		url.Values{"url": {url_visited},
			"content_type": {content_type},
			"body":         {body},
		})
	// TODO: log errors
}

func SaveCopyToHarken(resp *http.Response,
	ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil {
		return resp
	}

	length := resp.ContentLength
	if length <= 0 {
		length = 1024
	}
	buf := bytes.NewBuffer(make([]byte, 0, length))
	resp.Body = &BodyHandler{resp.Body, buf, resp}

	return resp
}

func TextButNotCode() goproxy.RespCondition {
	return goproxy.RespConditionFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "text/") {
				return false
			}
			r := strings.HasPrefix(contentType, "text/css") ||
				strings.HasPrefix(contentType, "text/javascript") ||
				strings.HasPrefix(contentType, "text/json")
			return !r
		})
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
	return goproxy.RespConditionFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
			if resp == nil {
				return false
			}
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
	// read the config file
	var configfile string
	flag.StringVar(&configfile, "config", "./config.json", "JSON config file")
	flag.Parse()

	file, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Fatal(err)
	}

	f := ConfigData{}
	err = json.Unmarshal(file, &f)
	if err != nil {
		log.Fatal(err)
	}

	SUBMIT_URL = f.SubmitURL

	var host_blacklist = []*regexp.Regexp{}
	var full_blacklist = []*regexp.Regexp{}
	var path_suffix_blacklist = []string{}

	content, err := ioutil.ReadFile(f.DomainBlacklistFile)
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" {
				host_blacklist = append(host_blacklist, regexp.MustCompile(line))
			}
		}
	}

	content, err = ioutil.ReadFile(f.FullBlacklistFile)
	if err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if line != "" {
				full_blacklist = append(full_blacklist, regexp.MustCompile(line))
			}
		}
	}

	content, err = ioutil.ReadFile(f.SuffixBlacklistFile)
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
		TextButNotCode(),
		goproxy.Not(UrlSuffixMatches(path_suffix_blacklist...)),
		goproxy.Not(goproxy.ReqHostMatches(host_blacklist...)),
		goproxy.Not(UrlMatchesAny(full_blacklist...)),
	).DoFunc(SaveCopyToHarken)
	log.Fatal(http.ListenAndServe("localhost:8080", proxy))
}
