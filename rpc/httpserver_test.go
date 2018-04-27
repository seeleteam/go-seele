/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	whiteList_all     = []string{"*"}
	whiteList_domains = []string{"seele.com", "www.test.com"}
)

func Test_WhiteList(t *testing.T) {
	testWhiteList(t, whiteList_all, "http://sometest.com", true)
	testWhiteList(t, whiteList_all, "http://www.baidu.com", true)
	testWhiteList(t, whiteList_all, "http://www.baidu.com:8080", true)
	testWhiteList(t, nil, "http://www.baidu.com", true)
	testWhiteList(t, whiteList_domains, "http://www.baidu.com", false)
	testWhiteList(t, whiteList_domains, "http://www.test.com", true)
	testWhiteList(t, whiteList_domains, "http://www.test.com:1234", true)
	testWhiteList(t, whiteList_domains, "http://127.0.0.1", true)
	testWhiteList(t, whiteList_domains, "http://seele.com/test/666", true)
}

func testWhiteList(t *testing.T, list []string, host string, expected bool) {
	_, filter := NewHTTPServer(list, nil)
	req := httptest.NewRequest(http.MethodPost, host, strings.NewReader(""))
	req.Header.Set("content-type", "application/json")
	if isValid := filter.isValideHost(req); isValid != expected {
		t.Fatalf("hostFilter test failed, host: %s", host)
	}
}

func Test_HTTPServe(t *testing.T) {
	serve, _ := NewHTTPServer(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "http://url.com", strings.NewReader(""))
	req.Header.Set("content-type", "application/json")

	w := httptest.NewRecorder()

	serve.ServeHTTP(w, req)
	if w.Body.Len() == 0 {
		t.Fatalf("HTTPServe test failed")
	}

	serve, _ = NewHTTPServer(nil, nil)

	req = httptest.NewRequest(http.MethodPost, "http://url.com", strings.NewReader(""))
	req.Header.Set("content-type", "application/json")

	w = httptest.NewRecorder()

	serve.ServeHTTP(w, req)
	if w.Body.Len() != 0 {
		t.Fatalf("HTTPServe test failed")
	}
}
