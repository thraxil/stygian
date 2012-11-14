package main

import (
	"fmt"
	"github.com/elazarl/goproxy"
	"log"
	"net/http"
	"strings"
)

func filter(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	// ignore non 200/304s
	if resp.StatusCode != 200 && resp.StatusCode != 304 {
		return resp
	}
	if resp.StatusCode == 200 {
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "text/") {
			if strings.HasPrefix(contentType, "text/css") || 
				strings.HasPrefix(contentType, "text/javascript") {
				return resp
			}
			// full response with a body. save it, index it, etc.
			fmt.Println("200", contentType, resp.ContentLength, resp.Request.URL)
			// log()
			// save()
			// index()
		}
	} else if resp.StatusCode == 304 {
		// just a 304. don't need to re-index or save
		// just log it. How to filter out only text/* types though?
		// 304s don't include Content-Type headers...
		fmt.Println("304", resp.Request.URL)
		// log()
	}
	return resp
}

func main() {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false
	proxy.OnResponse().DoFunc(filter)
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
