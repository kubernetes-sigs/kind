package images

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
)

const DockerHubAuthURLBase = "https://auth.docker.io/token"
const DockerHubAPIURLBase = "https://registry.hub.docker.com/v2/"

type DockerHubAuthResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // It's always 300 (sec) at this moment
}

type DockerHubTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func constructDockerHubAuthURL(image string) (string, error) {
	u, err := url.Parse(DockerHubAuthURLBase)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("service", "registry.docker.io")
	q.Set("scope", fmt.Sprintf("repository:%s:pull", image))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func constructDockerHubAPIURL(image string) (string, error) {
	u, err := url.Parse(DockerHubAPIURLBase)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, image, "tags/list")

	return u.String(), nil
}

func retrieveDockerHubAuthToken(image string) (string, error) {
	url, err := constructDockerHubAuthURL(image)
	if err != nil {
		return "", err
	}

	var body string
	if os.Getenv("DOCKERHUB_USERNAME") != "" && os.Getenv("DOCKERHUB_PASSWORD") != "" {
		body, err = httpGet(url, "", &BasicAuthInfo{
			Username: os.Getenv("DOCKERHUB_USERNAME"),
			Password: os.Getenv("DOCKERHUB_PASSWORD"),
		})
	} else {
		body, err = httpGet(url, "", nil)
	}
	if err != nil {
		return "", err
	}

	var resp DockerHubAuthResponse

	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return "", err
	}

	return resp.AccessToken, nil
}

func retrieveFromDockerHub(image string) ([]string, error) {
	dockerHubAccessToken, err := retrieveDockerHubAuthToken(image)
	if err != nil {
		return nil, err
	}

	dockerHubAPIURL, err := constructDockerHubAPIURL(image)
	if err != nil {
		return nil, err
	}

	body, err := httpGet(dockerHubAPIURL, dockerHubAccessToken, nil)
	if err != nil {
		return nil, err
	}

	var resp DockerHubTagsResponse

	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, err
	}

	tags := resp.Tags

	// Reverse the order of the tags to make it ordered as: "latest => oldest"
	for i, j := 0, len(tags)-1; i < j; i, j = i+1, j-1 {
		tags[i], tags[j] = tags[j], tags[i]
	}

	return tags, nil
}
