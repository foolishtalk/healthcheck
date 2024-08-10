package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var config ServiceConfig

// CheckServiceStatus sends an HTTP GET request to the specified URL and checks if the service is up.
func CheckServiceStatus(url string) (bool, error) {
	client := http.Client{
		Timeout: 5 * time.Second, // Set a timeout for the request
	}
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check if the status code is 2xx (success)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	return false, fmt.Errorf("service returned non-2xx status code: %d", resp.StatusCode)
}

func parseSerivcesJSON() {
	// read local file
	jsonFile, err := os.Open("services.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Printf(err.Error())
	}

	defer func(jsonFile *os.File) {
		err := jsonFile.Close()
		if err != nil {
			fmt.Printf(err.Error())
		}
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		fmt.Printf(err.Error())
	}
}

func main() {

	parseSerivcesJSON()

	for i := 0; i < len(config.URLs); i++ {
		url := config.URLs[i]
		status, err := CheckServiceStatus(url)
		if status {
			return
		}
		if err != nil {
			wecomNotify("服务异常，请求超时："+url+err.Error(), config.Wecom_hook_url)
			fmt.Printf("Error checking service status: %v  %v\n", err, url)
		} else {
			wecomNotify("服务异常，请求失败："+url, config.Wecom_hook_url)
			fmt.Printf("Service is down:%v\n", url)
		}
	}
}
