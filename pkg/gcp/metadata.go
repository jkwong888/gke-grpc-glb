package gcp

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
)

func GetMetaData(ctx context.Context, path string) *string {
	metaDataURL := "http://metadata/computeMetadata/v1/"
	req, _ := http.NewRequest(
		"GET",
		metaDataURL+path,
		nil,
	)
	req.Header.Add("Metadata-Flavor", "Google")
	req = req.WithContext(ctx)
	code, body := makeRequest(req)

	if code == 200 {
		bodyStr := string(body)
		return &bodyStr
	}

	return nil
}

func makeRequest(r *http.Request) (int, []byte) {
	//transport := http.Transport{DisableKeepAlives: true}
	//octr := &ochttp.Transport{}
	//client := &http.Client{Transport: octr}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		message := "Unable to call backend: " + err.Error()
		log.Printf(message)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		message := "Unable to read response body: " + err.Error()
		log.Printf(message)
	}

	return resp.StatusCode, body
}