package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

func wecomNotify(content string, url string) {
	payload := []byte(`
    {
      "msgtype": "text",
      "text": {
          "content": "` + content + `"
      }
    }
    `)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("request failed:", err)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("response fail:" + err.Error())
		return
	}
	log.Println("response:", string(body))
}
