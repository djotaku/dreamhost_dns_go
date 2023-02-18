package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

type UrlIPPair struct {
	url       string
	ipAddress string
}

type innerData struct {
	Records []map[string]string `json:"data"`
}

// commandResult for when you only care about the result
type commandResult struct {
	Data string `json: "result"`
}

// webGet handles contacting a URL
func webGet(url string) string {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		log.Fatalf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
	}
	if err != nil {
		log.Fatal(err)
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
	apiURLBase := "https://api.dreamhost.com"
	substring := "/?key=" + apiKey + "&cmd=" + command + "&format=json"
	fullURL := apiURLBase + substring
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

// addDNSRecord adds an IP address to a domain in dreamhost
func addDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-add_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	fmt.Printf("Result of trying to add DNS record is %s\n", result.Data)
	return result.Data
}

// deleteDNSRecord deletes an IP address to a domain in dreamhost
func deleteDNSRecord(domain string, newIPAddress string, apiKey string) string {
	command := "dns-remove_record&record=" + domain + "&type=A" + "&value=" + newIPAddress
	response := submitDreamhostCommand(command, apiKey)
	var result commandResult
	err := json.Unmarshal([]byte(response), &result)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	fmt.Printf("Result of trying to delete DNS record is %s\n", result.Data)
	return result.Data
}

func updateDNSRecord(domain string, currentIP string, newIPAddress string, apiKey string){
  resultOfAdd := addDNSRecord(domain, newIPAddress, apiKey)
  if resultOfAdd == "sucess"{
    deleteDNSRecord(domain, currentIP, apiKey)
  }
}

func main() {
	newIPAddress := getHostIpAddress()
	settingsJson, err := os.Open("settings.json")
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	defer func(settingsJson *os.File) {
		err := settingsJson.Close()
		if err != nil {

		}
	}(settingsJson)
	byteValue, _ := ioutil.ReadAll(settingsJson)
	var settings *credentials
	err = json.Unmarshal(byteValue, &settings)
	if err != nil {
		fmt.Printf("could not unmarshal json: %s\n", err)
		return
	}
	fmt.Printf("IP address outside the NAT is: %s\n", newIPAddress)
	fmt.Printf("Dreamhost API key is: %s\n", settings.ApiKey)
	fmt.Printf("Domains to update are: %s\n", settings.Domains)
	dnsRecords := getDNSRecords(settings.ApiKey)
	var records dnsRecordsJSON
	err = json.Unmarshal([]byte(dnsRecords), &records)
	if err != nil {
		fmt.Printf("err is: %s\n", err)
	}
	var domainDNSIPPairs []UrlIPPair
	for _, url := range records.Data {
		if contains(settings.Domains, url["record"]) {
			pair := UrlIPPair{url: url["record"], ipAddress: url["value"]}
			domainDNSIPPairs = append(domainDNSIPPairs, pair)
		}
	}
	for _, domanToUpdate := range domainDNSIPPairs {
		currentIP := domanToUpdate.ipAddress
		domain := domanToUpdate.url
		if currentIP != newIPAddress {
			updateDNSRecord(domain, currentIP, newIPAddress, settings.ApiKey)
		}
	}
  // last thing for parity with my Python one - adding in new domains if there are new domains in the settings
}
