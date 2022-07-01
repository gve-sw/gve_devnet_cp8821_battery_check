/*
Copyright (c) 2022 Cisco and/or its affiliates.
This software is licensed to you under the terms of the Cisco Sample
Code License, Version 1.1 (the "License"). You may obtain a copy of the
License at
               https://developer.cisco.com/docs/licenses
All use of the material herein must be in accordance with the terms of
the License. All rights not expressly granted by the License are
reserved. Unless required by applicable law or agreed to separately in
writing, software distributed under the License is distributed on an "AS
IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
or implied.
*/
package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var inputfile string
var chkTemp float64
var timeout int
var vlog bool
var good int
var bad int
var unreachable int
var hightemp int

const workers = 10

type BatteryInfo struct {
	ip     string
	health string
	temp   string
}

func main() {
	// Generate timestamp
	currentTime := time.Now()
	timestamp := string(currentTime.Format("20060102-150405"))

	// parse command line arguments
	flag.StringVar(&inputfile, "infile", "", "Text list of IP addresses to check (required)")
	flag.Float64Var(&chkTemp, "temp", 50, "High temperature threshold in C (default 50)")
	flag.IntVar(&timeout, "timeout", 10, "Time to wait for response from remote IP Phone in seconds (default 10)")
	flag.BoolVar(&vlog, "v", false, "Enable verbose logging")

	flag.Usage = func() {
		fmt.Println("Usage:")
		flag.PrintDefaults()
	}
	// Check that input file was provided - else print usage info
	flag.Parse()
	if inputfile == "" {
		fmt.Println("Please provide an input file!")
		flag.Usage()
		os.Exit(1)
	}

	// Open input file
	infile, err := os.Open(inputfile)
	if err != nil {
		log.Fatal(err)
	}
	defer infile.Close()

	// Count addresses in file:
	fmt.Println("Validating input file...")
	validAddr, invalidAddr := countLines(infile)
	fmt.Println("Found " + strconv.Itoa(validAddr) + " addresses to check")
	if invalidAddr >= 1 {
		fmt.Println(strconv.Itoa(invalidAddr) + " adddreses are invalid & will not be checked.")
	}
	// Reset to first line in file after reading during line count
	infile.Seek(0, io.SeekStart)

	// Create output files
	allResults, err := os.Create(timestamp + "-ALL.csv")
	if err != nil {
		log.Fatal(err)
	}
	badResults, err := os.Create(timestamp + "-BAD.csv")
	if err != nil {
		log.Fatal(err)
	}
	// Write header rows to each file
	allResults.WriteString("IP Address, Battery Health, Battery Temp\n")
	badResults.WriteString("IP Address, Battery Health, Battery Temp\n")
	defer allResults.Close()
	defer badResults.Close()

	// Create channels for jobs queue & worker results
	jobs := make(chan string, validAddr)
	results := make(chan BatteryInfo, workers)

	// Start workers
	var wg sync.WaitGroup
	for w := 1; w <= workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			getWebPage(w, jobs, results)
		}()
	}

	// Read each line of input file, and send to getWebPage
	fmt.Println("Working...")
	scanner := bufio.NewScanner(infile)
	for scanner.Scan() {

		// Strip any whitespace from IP
		ip := strings.TrimSpace(scanner.Text())
		// Ensure IP is valid
		if net.ParseIP(strings.Split(ip, ":")[0]) == nil {
			if vlog {
				fmt.Println("Invalid address: ", ip)
			}
			continue
		}
		// Load into jobs queue
		jobs <- ip
	}
	if vlog {
		fmt.Println("All jobs loaded into queue!")
	}
	// Close jobs channel after loading everything in
	close(jobs)

	for a := 1; a <= validAddr; a++ {
		battery_status := <-results
		if vlog {
			fmt.Println("Got Result, writing to CSV: ", battery_status)
		}
		// Write line to file
		result_info := fmt.Sprintf("%s,%s,%s\n", battery_status.ip, battery_status.health, battery_status.temp)
		_, err := allResults.WriteString(result_info)
		if err != nil {
			log.Fatal(err)
		}
		// Increment counters for result summary
		if battery_status.health == "Good" {
			good += 1
		} else {
			bad += 1
		}
		// If battery status is anything except "Good",
		// it gets added to the "bad" list
		if !strings.Contains(battery_status.health, "Good") {
			_, err = badResults.WriteString(result_info)
			if err != nil {
				log.Fatal(err)
			}
		}
		// If we got a temp from the IP Phone,
		// check against temp threshold
		if battery_status.health != "Unknown" {
			// Split temp string & pull digits out
			temp := strings.Split(battery_status.temp, " degrees Celsius")[0]
			// Convert to Float & check against provided threshold
			if a, err := strconv.ParseFloat(temp, 64); a > chkTemp {
				if err != nil {
					continue
				}
				hightemp += 1
			}
		}
		// Print worker status
		if vlog {
			fmt.Println("Jobs remaining:   " + strconv.Itoa(len(jobs)) + " of " + strconv.Itoa(cap(jobs)))
			fmt.Println("Results in queue:", strconv.Itoa(len(results)))
		}

	}
	// Wait for workers to finish

	wg.Wait()
	// Print results summary
	fmt.Println("Done!")
	fmt.Println("Summary: ")
	fmt.Println(" - Good: " + strconv.Itoa(good))
	fmt.Println(" - Bad: " + strconv.Itoa(bad))
	fmt.Println("Breakdown of 'bad' status:")
	fmt.Println(" - Health: " + strconv.Itoa(bad-(hightemp+unreachable)))
	fmt.Println(" - High Temp: " + strconv.Itoa(hightemp))
	fmt.Println(" - Unreachable/Unknown: " + strconv.Itoa(unreachable))

}

// getWebPage queries a remote web page
func getWebPage(id int, addresses <-chan string, results chan<- BatteryInfo) {
	for address := range addresses {
		if vlog {
			fmt.Println("Worker ID:" + strconv.Itoa(id) + " - Working on : " + address)
		}
		// HTTP Client Config.
		// Disable Certificate check & Set timeout
		tlsCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: tlsCfg,
		}
		// Send HTTP GET
		resp, err := client.Get("https://" + address)
		// Fallback to HTTP if client does not support TLS
		if err != nil && strings.HasSuffix(err.Error(), "server gave HTTP response to HTTPS client") {
			if vlog {
				fmt.Println("Worker ID:" + strconv.Itoa(id) + " | Client: " + address + " - Fallback to HTTP")
			}
			resp, err = client.Get("http://" + address)
		}
		if err != nil {
			if vlog {
				fmt.Println("Worker ID:" + strconv.Itoa(id) + " - Cannot connect to: " + address)
				fmt.Println(err)
			}
			unreachable += 1
			results <- BatteryInfo{ip: address, health: "Unknown", temp: ""}
			continue
		}
		if vlog {
			fmt.Println("Worker ID:" + strconv.Itoa(id) + " Got response from " + address)
		}
		// Parse HTML response
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		info := new(BatteryInfo)
		info.ip = address
		// Find table on IP Phone home page, which contains health stats / info
		doc.Find("table").Each(func(index int, tablehtml *goquery.Selection) {
			// Battery info is located in third table
			if index == 2 {
				// Locate table rows that contain battery health & temp info
				tablehtml.Find("tr").Each(func(index int, tablerow *goquery.Selection) {
					if strings.Contains(tablerow.Text(), "Battery health") {
						info.health = strings.Split(tablerow.Text(), "Battery health")[1]
					}
					if strings.Contains(tablerow.Text(), "Battery temperature:") {
						info.temp = strings.Split(tablerow.Text(), "Battery temperature: ")[1]
					}
				})
			}
		})
		resp.Body.Close()
		results <- *info
	}
}

// countLines takes in a file & counts the number of lines which contain a valid IPv4 address
func countLines(input *os.File) (int, int) {
	// Read file
	scanner := bufio.NewScanner(input)
	valid := 0
	invalid := 0

	// Count lines
	for scanner.Scan() {
		ip := strings.TrimSpace(scanner.Text())
		// Ensure IP is valid
		if net.ParseIP(strings.Split(ip, ":")[0]) != nil {
			valid++
		} else {
			invalid++
		}
	}
	return valid, invalid
}
