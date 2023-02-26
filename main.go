package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/adrg/xdg"
)

//credentials are the credentials needed to talk to the Dreamhost API
type credentials struct {
	ApiKey  string `json:"api_key"`
	Domains []string
}

//dnsRecordsJSON holds the JSON returned by the Dreamhost API
type dnsRecordsJSON struct {
	Data []map[string]string `json:"data"`
}

type urlIPPair struct {
	url       string
	ipAddress string
}

type innerData struct {
	Records []map[string]string `json:"data"`
}

// commandResult for when you only care about the result
type commandResult struct {
	Data string `json:"result"`
}

// webGet handles contacting a URL
func webGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
    statusCodeString := fmt.Sprintf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
    log.Println(statusCodeString)
	}
	if err != nil {
		log.Println(err)
	}
	return string(result)
}

//getHostIpAddress gets the outside IP address of the computer it's on
func getHostIpAddress() string {
	ipAddress := webGet("https://api.ipify.org")
	return string(ipAddress)
}

//submitDreamhostCommand takes in a command string and api key, contacts the API and returns the result
func submitDreamhostCommand(command string, apiKey string) string {
	apiURLBase := "https://api.dreamhost.com/?"
	queryParameters := url.Values{}
	queryParameters.Set("key", apiKey)
	queryParameters.Add("cmd", command)
	queryParameters.Add("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	dreamhostResponse := webGet(fullURL)
	return dreamhostResponse
}

//getDNSRecords gets the DNS records from the Dreamhost API
func getDNSRecords(apiKey string) string {
	dnsRecords := submitDreamhostCommand("dns-list_records", apiKey)
	return dnsRecords
}

// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

//conditionalLog will print a log to the console if logActive true
func conditionalLog(message string, logActive bool){
  if logActive{
    log.Println(message)
  }
}

// addDNSRecord adds an IP address to a domain in dreamhost
func addDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-add_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	log.Printf("Result of trying to add DNS record for %s is %s\n", domain, result.Data)
	return result.Data
}

// deleteDNSRecord deletes an IP address to a domain in dreamhost
func deleteDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-remove_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		log.Printf("Error: %s\n", err)
	}
	log.Printf("Result of trying to delete DNS record for %s is %s\n", domain, result.Data)
	return result.Data
}

func updateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string) {
	resultOfAdd := addDNSRecord(domain, newIPAddress, apiKey)
	if resultOfAdd == "sucess" {
		deleteDNSRecord(domain, currentIP, apiKey)
	}
}

func main() {

	logFilePath, _ := xdg.DataFile("dreamhostdns/dnsupdates.log")
	// once you figure out how to import https://github.com/natefinch/lumberjack/tree/v2.0 , use that
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0775)
	if err != nil {
		log.Printf("Error %s\n", err)
	}
	fileLogger := log.New(logFile, "", log.LstdFlags)

	// parse CLI flags
	verbose := flag.Bool("v", false, "prints log output to the commandline.")
	flag.Parse()

	// user wants verbose logs
	if *verbose {
		fmt.Println("User chose verbose CLI output!") // this is just a placeholder
	}

	configFilePath, err := xdg.ConfigFile("dreamhostdns/settings.json")
	if err != nil {
    conditionalLog(err.Error(), *verbose)
		fileLogger.Fatal(err)
	}
	fmt.Printf("Looking for settings.jon should be at the following path: %s\n", configFilePath)
	newIPAddress := getHostIpAddress()
	settingsJson, err := os.Open(configFilePath)
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println("Unable to open the confix file. Did you place it in the right spot?")
    conditionalLog(err.Error(), *verbose)
    fileLogger.Fatal(err)
	}
	defer func(settingsJson *os.File) {
		err := settingsJson.Close()
		if err != nil {
      errorString := fmt.Sprintf("Couldn't close the settings file. Error: %s", err)
			conditionalLog(errorString, *verbose)
      fileLogger.Fatal(errorString)

		}
	}(settingsJson)
	byteValue, _ := ioutil.ReadAll(settingsJson)
	var settings *credentials
	err = json.Unmarshal(byteValue, &settings)
	if err != nil {
		fmt.Println("Check that you do not have errors in your JSON file.")
    errorString := fmt.Sprintf("Could not unmashal json: %s\n", err)
    conditionalLog(errorString, *verbose)
    fileLogger.Fatal(errorString)
		return
	}
	fmt.Printf("IP address outside the NAT is: %s\n", newIPAddress)
	fileLogger.Printf("IP address outsite the NAT is %s\n", newIPAddress)
  fmt.Printf("Domains to update are: %s\n", settings.Domains)
	dnsRecords := getDNSRecords(settings.ApiKey)
	var records dnsRecordsJSON
	err = json.Unmarshal([]byte(dnsRecords), &records)
	if err != nil {
    errorString := fmt.Sprintf("Unable to unmashal the JSON from Dreamhost. err is: %n", err)
    conditionalLog(errorString, *verbose)
    fileLogger.Fatal(errorString)
	}

	var updatedDomains []string
	for _, url := range records.Data {
		if contains(settings.Domains, url["record"]) {
			currentDomain := urlIPPair{url: url["record"], ipAddress: url["value"]}
			updatedDomains = append(updatedDomains, currentDomain.url)
			if currentDomain.ipAddress != newIPAddress {
        logString := fmt.Sprintf("%s has an old IP of %s. Will attempt to change to %s", currentDomain.url, currentDomain.ipAddress, newIPAddress)
        fileLogger.Printf(logString)
        conditionalLog(logString, *verbose)
				updateDNSRecord(currentDomain.url, currentDomain.ipAddress, newIPAddress, settings.ApiKey)
			} else {
        logString := fmt.Sprintf("%s is already set to IP address: %s", currentDomain.url, currentDomain.ipAddress)
        fileLogger.Println(logString)
        conditionalLog(logString, *verbose)
			}

		}
	}
	// add in new domains that weren't already in Dreamhost (also handles accidentally deleted domains)
	for _, domain := range settings.Domains {
		if !contains(updatedDomains, domain) {
			addDNSRecord(domain, newIPAddress, settings.ApiKey)
		}
	}
}
