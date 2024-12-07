package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

var checkTemplate = template.Must(template.ParseFiles("templates/check.html"))

func main() {
	mux := buildMux()

	port := os.Getenv("PORT")
	addr := "0.0.0.0:" + port

	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, mux)
	log.Fatal(http.ListenAndServe(addr, loggedRouter))
}

func buildMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.HandleFunc("/up", upHandler)
	mux.HandleFunc("/check", checkAndReportHTML)
	mux.HandleFunc("/check.txt", checkAndReport)
	return mux
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
		// Ignore invalid URLs (like in the header row)
		if isURL(check.URL) {
			result = append(result, check)
		}
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

func isURL(s string) bool {
	hasHttpPrefix := strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
	if !hasHttpPrefix {
		return false
	}

	_, err := url.Parse(s)
	return err == nil
}

// CheckResult contains an array of checks and helper function to return
// - the number of passed checks
// - allChecksOk is true if all checks passed
// - the slowest check
// - list of failed checks
type CheckResult struct {
	Checks []check
}

func (cr *CheckResult) SortChecks() {
	// sort Checks by ID (ascending)
	slices.SortFunc(cr.Checks, func(a, b check) int {
		return strings.Compare(strings.ToLower(a.ID), strings.ToLower(b.ID))
	})
}

func (cr *CheckResult) PassedChecks() int {
	passedChecks := 0
	for _, check := range cr.Checks {
		if check.OK {
			passedChecks++
		}
	}
	return passedChecks
}
func (cr *CheckResult) AllChecksOk() bool {
	return cr.PassedChecks() == len(cr.Checks)
}
func (cr *CheckResult) SlowestCheck() *check {
	slowestCheck := cr.Checks[0]
	for _, check := range cr.Checks {
		if check.runtime > slowestCheck.runtime {
			slowestCheck = check
		}
	}
	return &slowestCheck
}
func (cr *CheckResult) FailedChecks() []check {
	var failedChecks []check
	for _, check := range cr.Checks {
		if !check.OK {
			failedChecks = append(failedChecks, check)
		}
	}
	return failedChecks
}

func runAllChecks(checks []check) CheckResult {
	channel := make(chan check)
	for _, check := range checks {
		go runSingleCheck(check, channel)
	}

	result := make([]check, len(checks))
	for i := 0; i < len(checks); i++ {
		check := <-channel
		result[i] = check
	}

	return CheckResult{Checks: result}
}

type ResultPageData struct {
	ErrorMessage string
	Checks       []check

	PassedChecks int
	TotalChecks  int
}

func checkAndReportHTML(res http.ResponseWriter, r *http.Request) {
	checkURL := r.FormValue("checks")
	if checkURL == "" {
		errorMessage := "ERROR: checks parameter missing\n"
		checkTemplate.Execute(res, ResultPageData{ErrorMessage: errorMessage})
		return
	}

	resp, err := http.Get(checkURL)
	if err != nil {
		errorMessage := "ERROR: Could not fetch checks CSV file\n"
		checkTemplate.Execute(res, ResultPageData{ErrorMessage: errorMessage})
		return
	}
	defer resp.Body.Close()

	checks := readChecksCSV(resp.Body)
	result := runAllChecks(checks)
	result.SortChecks()

	page := ResultPageData{
		Checks:       result.Checks,
		PassedChecks: result.PassedChecks(),
		TotalChecks:  len(checks),
	}

	err = checkTemplate.Execute(res, page)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func checkAndReport(res http.ResponseWriter, r *http.Request) {
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
	result := runAllChecks(checks)
	allChecksOk := result.AllChecksOk()
	slowestCheck := result.SlowestCheck()

	if allChecksOk {
		io.WriteString(res, fmt.Sprintf("%d checks OK\n", len(checks)))
		io.WriteString(res, "\n")

		message := fmt.Sprintf("Slowest %s:%v", slowestCheck.ID, slowestCheck.runtime)
		io.WriteString(res, message)
	} else {
		failures := result.FailedChecks()
		// Concatenate all URLs of failed checks
		var failedURLs []string
		for _, check := range failures {
			failedURLs = append(failedURLs, check.URL)
		}
		errorMessage := "ERROR: \n" + strings.Join(failedURLs, "\n")
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
		body, err := io.ReadAll(resp.Body)
		bodyText := string(body)

		if (err == nil) && strings.Contains(bodyText, check.ExpectedText) {
			check.OK = true
		}
	}

	check.runtime = time.Since(start)
	fmt.Printf("Check completed: %v+\n", check)
	channel <- check
}

// /up is a simple health check endpoint (used by kamal deploy)
func upHandler(res http.ResponseWriter, r *http.Request) {
	io.WriteString(res, "OK\n")
}
