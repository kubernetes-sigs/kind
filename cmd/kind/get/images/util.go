package images

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type BasicAuthInfo struct {
	Username string
	Password string
}

func httpGet(url, apiToken string, basicAuth *BasicAuthInfo) (string, error) {
	req, _ := http.NewRequest("GET", url, nil)
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}

	if basicAuth != nil {
		req.SetBasicAuth(basicAuth.Username, basicAuth.Password)
	}

	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error!\nURL: %s\nstatus code: %d\nbody:\n%s\n", url, resp.StatusCode, string(b))
	}

	return string(b), nil
}
