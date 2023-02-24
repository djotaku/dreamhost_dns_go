package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os
	"flag"

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
		log.Printf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
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

	// parse CLI flags
	verbose := flag.Bool("-v", false, "prints log output to the commandline.")
	flag.Parse()

	configFilePath, err := xdg.ConfigFile("dreamhostdns/settings.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Looking for settings.jon should be at the following path: %s\n", configFilePath)
	newIPAddress := getHostIpAddress()
	settingsJson, err := os.Open(configFilePath)
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println("Unable to open the confix file. Did you place it in the right spot?")
		log.Fatal(err)
	}
	defer func(settingsJson *os.File) {
		err := settingsJson.Close()
		if err != nil {
			log.Printf("Couldn't close the settings file. Error: %s", err)

		}
	}(settingsJson)
	byteValue, _ := ioutil.ReadAll(settingsJson)
	var settings *credentials
	err = json.Unmarshal(byteValue, &settings)
	if err != nil {
		fmt.Println("Check that you do not have errors in your JSON file.")
		log.Fatalf("could not unmarshal json: %s\n", err)
		return
	}
	fmt.Printf("IP address outside the NAT is: %s\n", newIPAddress)
	fmt.Printf("Domains to update are: %s\n", settings.Domains)
	dnsRecords := getDNSRecords(settings.ApiKey)
	var records dnsRecordsJSON
	err = json.Unmarshal([]byte(dnsRecords), &records)
	if err != nil {
		log.Fatalf("Unable to unmarshall JSON from Dreamhost. err is: %s\n", err)
	}

	var updatedDomains []string
	for _, url := range records.Data {
		if contains(settings.Domains, url["record"]) {
			currentDomain := urlIPPair{url: url["record"], ipAddress: url["value"]}
			updatedDomains = append(updatedDomains, currentDomain.url)
			if currentDomain.ipAddress != newIPAddress {
				log.Printf("%s has an old IP of %s. Will attempt to change to %s", currentDomain.url, currentDomain.ipAddress, newIPAddress)
				updateDNSRecord(currentDomain.url, currentDomain.ipAddress, newIPAddress, settings.ApiKey)
			} else {
				log.Printf("%s is already set to the IP address: %s", currentDomain.url, currentDomain.ipAddress)
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
