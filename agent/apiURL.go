package agent

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	pubURL     = "https://api.github.com"
	apiVersion = "/v3"
)

// APIRoot returns the root of a given API URL
// so for public github:
//   https://api.github.com/repos/my-org/my-repo/pulls/1 would return https://api.github.com
// and for enterprise github:
//   https://github.my-domain/api/v3/repos/my-org/my-repo/pulls/1 returns https://github.my-domain/api/v3
func APIRoot(repoURL string) (APIURL string, err error) {
	if strings.Contains(repoURL, pubURL) {
		APIURL = pubURL
		return
	}

	url, err := url.Parse(repoURL)
	if url == nil || err != nil {
		err = fmt.Errorf("Cannot parse repoURL %v", err)
		return
	}
	re := regexp.MustCompile("^.*/v3")
	match := re.FindString(url.Path)
	if match != "" {
		url.Path = match
	} else {
		err = fmt.Errorf("API URL is not version 3")
		return
	}
	APIURL = url.String()
	return
}
