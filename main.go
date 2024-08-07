package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/adrg/xdg"
	"github.com/djotaku/dreamhostapi/v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

// credentials are the credentials needed to talk to the Dreamhost API
type credentials struct {
	ApiKey  string `json:"api_key"`
	Domains []string
}

// conditionalLog will print a log to the console if logActive true
func conditionalLog(message string, logActive bool) {
	if logActive {
		log.Println(message)
	}
}

// getHostIpAddress gets the outside IP address of the computer it's on
func getHostIpAddress(logActive bool) string {
	ipAddress, _, err := dreamhostapi.WebGet("https://api.ipify.org")
	if err != nil {
		logMessage := fmt.Sprintf("Error getting IP address, cannot continue. Error: %s", err)
		fmt.Println(logMessage)
		conditionalLog(logMessage, logActive)
		log.Fatal(logMessage)
	}
	return string(ipAddress)
}

func getSettings(fileLogger *log.Logger, verbose *bool) *credentials {
	configFilePath, err := xdg.ConfigFile("dreamhostdns/settings.json")
	if err != nil {
		conditionalLog(err.Error(), *verbose)
		fileLogger.Fatal(err)
	}
	fmt.Printf("Looking for settings.jon. The file should be at the following path: %s\n", configFilePath)

	settingsJson, err := os.Open(configFilePath)
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println("Unable to open the config file. Did you place it in the right spot?")
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
	byteValue, _ := io.ReadAll(settingsJson)
	var settings *credentials
	err = json.Unmarshal(byteValue, &settings)
	if err != nil {
		fmt.Println("Check that you do not have errors in your JSON file.")
		errorString := fmt.Sprintf("Could not unmashal json: %s\n", err)
		conditionalLog(errorString, *verbose)
		fileLogger.Fatal(errorString)
	}
	return settings
}

func main() {

	logFilePath, _ := xdg.DataFile("dreamhostdns/dnsupdates.log")
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0775)
	if err != nil {
		log.Printf("Error %s\n", err)
	}
	fileLogger := log.New(logFile, "", log.LstdFlags)
	fileLogger.SetOutput(&lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    1, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
	fmt.Printf("Logs can be found at: %s\n", logFilePath)
	fileLogger.Println("############ NEW SESSION ################")

	// parse CLI flags
	verbose := flag.Bool("v", false, "prints log output to the commandline.")
	flag.Parse()

	settings := getSettings(fileLogger, verbose)

	newIPAddress := getHostIpAddress(*verbose)
	fmt.Printf("IP address outside the NAT is: %s\n", newIPAddress)
	fileLogger.Printf("IP address outsite the NAT is %s\n", newIPAddress)
	fmt.Printf("Domains to update are: %s\n", settings.Domains)

	dnsRecords, err := dreamhostapi.GetDNSRecords(settings.ApiKey)
	if err != nil {
		logMessage := fmt.Sprintf("Cannot continue without DNS records. Error: %s", err)
		fmt.Println(logMessage)
		conditionalLog(logMessage, *verbose)
		log.Fatal(logMessage)
	}

	currentDNSValues := make(map[string]string)
	for _, record := range dnsRecords.Data {
		currentDNSValues[record.Record] = record.Value
	}

	successMessage := "The following domains successfully updated: "
	for _, myDomain := range settings.Domains {
		if currentDNSValues[myDomain] == newIPAddress {
			logString := fmt.Sprintf("%s is already set to IP address: %s", myDomain, newIPAddress)
			fileLogger.Println(logString)
			conditionalLog(logString, *verbose)
		} else {
			logString := fmt.Sprintf("%s has an IP of %s. (If no value listed, this is a new domain.) Will attempt to change to %s (or add in the new domain)", myDomain, currentDNSValues[myDomain], newIPAddress)
			fileLogger.Printf(logString)
			conditionalLog(logString, *verbose)
			addResult, deleteResult, err := dreamhostapi.UpdateDNSRecord(myDomain, currentDNSValues[myDomain], newIPAddress, settings.ApiKey, "")
			updateResults := fmt.Sprintf("%s: addResult: %s, deleteResult: %s", myDomain, addResult.Result, deleteResult.Result)
			conditionalLog(updateResults, *verbose)
			fileLogger.Printf(updateResults)
			if err != nil {
				logMessage := fmt.Sprintf("An error occurred during DNS update. Add result: %s. Delete result: %s. Error: %s", addResult.Result, deleteResult.Result, err)
				conditionalLog(logMessage, *verbose)
				log.Println(logMessage)
			} else {
				successMessage += fmt.Sprintf("%s, ", myDomain)
			}
		}
	}
	fmt.Println(successMessage)
}
