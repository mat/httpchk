package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestGetChecksWithMissingChecksParam(t *testing.T) {
	resp := doRequest("GET", "/", emptyBody())

	expectStatus(t, resp, 404)
	expectHeader(t, resp, "Content-Type", "text/plain; charset=utf-8")

	expectBodyContains(t, resp, "ERROR: checks parameter missing")
}

func TestGetChecksWithoutURL(t *testing.T) {
	resp := doRequest("GET", "/?checks=not-a-URL", emptyBody())

	expectStatus(t, resp, 503)
	expectHeader(t, resp, "Content-Type", "text/plain; charset=utf-8")

	expectBodyContains(t, resp, "ERROR: Could not fetch checks CSV file")
}

func TestGetTwoSimpleChecks(t *testing.T) {
	resp := doRequest("GET", "/?checks=https://raw.githubusercontent.com/mat/httpchk/master/checks.csv", emptyBody())

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "text/plain; charset=utf-8")

	expectBodyContains(t, resp, "2 checks OK")
	expectBodyMatches(t, resp, "Slowest .*:.*ms")
}

func doRequest(method, uri string, body *bytes.Buffer) *httptest.ResponseRecorder {
	return doRequestWithHeader(method, uri, body, nil)
}

func doRequestWithHeader(method, uri string, body *bytes.Buffer, header *http.Header) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		panic(err)
	}
	if header != nil {
		req.Header = *header
	}

	router := buildRouter()
	router.ServeHTTP(resp, req)
	return resp
}

func expectStatus(t *testing.T, resp *httptest.ResponseRecorder, expected int) {
	if status := resp.Code; status != expected {
		t.Errorf("wrong status code: is %v but wanted %v", status, expected)
	}
}

func expectErrorJSON(t *testing.T, resp *httptest.ResponseRecorder, expectedStatusCode int, expectedErrorText string) {
	expectStatus(t, resp, expectedStatusCode)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectedJSON := fmt.Sprintf(`{"error":"%s"}`, expectedErrorText)
	expectBodyContains(t, resp, expectedJSON)
}

func expectBodyContains(t *testing.T, resp *httptest.ResponseRecorder, expected string) {
	if !strings.Contains(resp.Body.String(), expected) {
		t.Errorf("wrong body: '%v' not contained in '%v'",
			expected, resp.Body.String())
	}
}

func expectBodyMatches(t *testing.T, resp *httptest.ResponseRecorder, expectedRegexp string) {
	var regex = regexp.MustCompile(expectedRegexp)
	body := resp.Body.String()
	if !regex.MatchString(body) {
		t.Errorf("wrong body: did not match regexp '%v': %v", expectedRegexp, body)
	}
}

func expectHeader(t *testing.T, resp *httptest.ResponseRecorder, headerName string, expected string) {
	if resp.Header().Get(headerName) != expected {
		t.Errorf("wrong header %v: is '%v' but wanted '%v'",
			headerName, resp.Header().Get(headerName), expected)
	}
}

func expectNoError(t *testing.T, e error) {
	if e != nil {
		t.Errorf("expected no error but got: %v", e)
	}
}

func expectIsTrue(t *testing.T, b bool) {
	if b != true {
		t.Errorf("expected b==true but was false")
	}
}

func expectSameString(t *testing.T, str1 string, str2 string) {
	if str1 != str2 {
		t.Errorf("expected same strings, but got: str1=%v and str2=%v", str1, str2)
	}
}

func expectEmptyHeader(t *testing.T, resp *httptest.ResponseRecorder, headerName string) {
	actualHeaderValue := resp.Header().Get(headerName)
	if len(actualHeaderValue) > 0 {
		t.Errorf("expected empty header for %v, but found '%v'",
			headerName, actualHeaderValue)
	}
}
func expectHeaderMatches(t *testing.T, resp *httptest.ResponseRecorder, headerName string, expectedRegexp string) {
	var regex = regexp.MustCompile(expectedRegexp)
	actualHeaderValue := resp.Header().Get(headerName)
	if !regex.MatchString(actualHeaderValue) {
		t.Errorf("wrong header %v: '%v' did not match '%v'",
			headerName, actualHeaderValue, expectedRegexp)
	}
}

func body(str string) *bytes.Buffer {
	return bytes.NewBufferString(str)
}
func emptyBody() *bytes.Buffer {
	return bytes.NewBufferString("hello")
}
