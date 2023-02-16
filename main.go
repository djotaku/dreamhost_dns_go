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

type credentials struct {
	ApiKey  string `json:"api_key"`
	Domains []string
}

type dnsRecordsJSON struct {
	Data map[string]map[string]string `json:"data"`
}

type innerData struct {
	Records []map[string]string `json:"data"`
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

func getDNSRecords(apiKey string) string {
	dnsRecords := submitDreamhostCommand("dns-list_records", apiKey)
	return dnsRecords
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
	fmt.Printf("DNS Records are: %s\n", dnsRecords)
	// var records *dnsRecordsJSON
	//records = &dnsRecordsJSON{
	//	Data: map[string]map[string]string{"something": "something"},
	//}
	//err = json.Unmarshal([]byte(dnsRecords), &records)
	// fmt.Printf("DNS Records are: %s\n", records.Data)
}
