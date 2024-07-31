package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	UpdateCommand = "curl -sS https://tsui.neuralink.com/install | bash"
	GithubRepo    = "neuralinkcorp/tsui"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// Fetches the latest version of tsui from GitHub releases.
func FetchLatestVersion() (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GithubRepo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	target := &githubRelease{}
	err = json.NewDecoder(resp.Body).Decode(target)
	if err != nil {
		return "", err
	}

	version := strings.TrimPrefix(target.TagName, "v")
	return version, nil
}
