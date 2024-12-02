package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
)

func main() {
	router := buildRouter()
	router.ServeFiles("/static/*filepath", http.Dir("public/static/"))

	port := os.Getenv("PORT")
	addr := "0.0.0.0:" + port

	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(addr, loggedRouter))
}

func buildRouter() *httprouter.Router {
	router := httprouter.New()
	router.GET("/up", upHandler)
	router.GET("/", checkAndReport)

	return router
}

func readChecksCSV(r io.ReadCloser) []check {
	csvReader := csv.NewReader(r)
	csvReader.TrimLeadingSpace = true
	csvReader.LazyQuotes = true

	var result []check
	for {
		fields, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		check := checkFromCSVFields(fields)
		result = append(result, check)
	}

	fmt.Printf("Read %d checks from CSV.\n", len(result))
	return result
}

func checkFromCSVFields(fields []string) check {
	return check{
		ID:           fields[0],
		URL:          fields[1],
		ExpectedText: fields[2],
	}
}

func runAllChecks(checks []check) (allChecksOk bool, failures string, slowestCheck *check) {
	channel := make(chan check)
	for _, check := range checks {
		go runSingleCheck(check, channel)
	}

	allChecksOk = true

	for i := 0; i < len(checks); i++ {
		check := <-channel
		if check.OK {
			fmt.Printf("Returned in %v: %s ok.\n", check.runtime, check.URL)
		} else {
			fmt.Fprintf(os.Stderr, "FAILED after %v: %s\n", check.runtime, check.URL)
		}
		if !check.OK {
			allChecksOk = false
			failures = failures + check.URL + "\n"
		}
		slowestCheck = &check
	}

	return allChecksOk, failures, slowestCheck
}

func checkAndReport(res http.ResponseWriter, r *http.Request, p httprouter.Params) {
	hoursParam := r.FormValue("hours")
	hours := strings.Split(hoursParam, ",")
	if len(hoursParam) > 0 && !contains(hours, time.Now().Hour()) {
		io.WriteString(res, fmt.Sprintf("%s\nnot running tests cause now is not the time (%d not included in hours=%v)\n", time.Now(), time.Now().Hour(), hours))
		return
	}

	checkURL := r.FormValue("checks")
	if checkURL == "" {
		errorMessage := "ERROR: checks parameter missing\n"
		http.Error(res, errorMessage, http.StatusNotFound)
		return
	}

	resp, err := http.Get(checkURL)
	if err != nil {
		errorMessage := "ERROR: Could not fetch checks CSV file\n"
		http.Error(res, errorMessage, http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	checks := readChecksCSV(resp.Body)
	allChecksOk, failures, slowestCheck := runAllChecks(checks)

	if allChecksOk {
		io.WriteString(res, fmt.Sprintf("%d checks OK\n", len(checks)))
		io.WriteString(res, "\n")

		message := fmt.Sprintf("Slowest %s:%v", slowestCheck.ID, slowestCheck.runtime)
		io.WriteString(res, message)
	} else {
		errorMessage := "ERROR: \n" + failures
		http.Error(res, errorMessage, http.StatusServiceUnavailable)
	}
}

func contains(ints []string, n int) bool {
	for _, str := range ints {
		i, _ := strconv.Atoi(str)
		if i == n {
			return true
		}
	}

	return false
}

type check struct {
	ID           string
	URL          string
	ExpectedText string
	OK           bool
	runtime      time.Duration
}

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

func runSingleCheck(check check, channel chan check) {
	check.OK = false

	start := time.Now()
	check.runtime = 0

	timeout := time.Duration(29 * time.Second)
	transport := http.Transport{
		Dial: timeoutDialer(timeout, timeout),
	}
	client := http.Client{
		Transport: &transport,
	}
	resp, err := client.Get(check.URL)
	if err == nil && resp.StatusCode == 200 {
		if check.ExpectedText == "" {
			check.OK = true
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		bodyText := string(body)

		if (err == nil) && strings.Contains(bodyText, check.ExpectedText) {
			check.OK = true
		}
	}

	check.runtime = time.Since(start)
	channel <- check
}

// /up is a simple health check endpoint (used by kamal deploy)
func upHandler(res http.ResponseWriter, r *http.Request, p httprouter.Params) {
	io.WriteString(res, "OK\n")
}
