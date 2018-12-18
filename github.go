// github.go - ham fisted accesses to github /user with caching.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// Maximum time an item is considered valid in the cache.
	maxAgeDays = 30
	cacheDir   = "/tmp/scoreboard.cache"
)

// GithubUserResponse holds information about a particular github user.
type GithubUserResponse struct {
	Login             string      `json:"login"`
	ID                int         `json:"id"`
	NodeID            string      `json:"node_id"`
	AvatarURL         string      `json:"avatar_url"`
	GravatarID        string      `json:"gravatar_id"`
	URL               string      `json:"url"`
	HTMLURL           string      `json:"html_url"`
	FollowersURL      string      `json:"followers_url"`
	FollowingURL      string      `json:"following_url"`
	GistsURL          string      `json:"gists_url"`
	StarredURL        string      `json:"starred_url"`
	SubscriptionsURL  string      `json:"subscriptions_url"`
	OrganizationsURL  string      `json:"organizations_url"`
	ReposURL          string      `json:"repos_url"`
	EventsURL         string      `json:"events_url"`
	ReceivedEventsURL string      `json:"received_events_url"`
	Type              string      `json:"type"`
	SiteAdmin         bool        `json:"site_admin"`
	Name              string      `json:"name"`
	Company           interface{} `json:"company"`
	Blog              string      `json:"blog"`
	Location          string      `json:"location"`
	Email             interface{} `json:"email"`
	Hireable          interface{} `json:"hireable"`
	Bio               interface{} `json:"bio"`
	PublicRepos       int         `json:"public_repos"`
	PublicGists       int         `json:"public_gists"`
	Followers         int         `json:"followers"`
	Following         int         `json:"following"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// githubUserInfo returns github information about a given username.
// A boolean flag is returned to mean "user not found in github". All
// other unexpected conditions return an error.
func githubUserInfo(username string) (GithubUserResponse, bool, error) {
	// Attempt to fetch json from cache first. Failing that,
	// make a request directly to github.
	jdata, ok, err := cached(username)
	if err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("error retrieving cached data: %v", err)
	}
	if !ok {
		r, err := http.Get("https://api.github.com/users/" + username)
		if err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("error retrieving github user: %v", err)
		}
		// Ignore users that are not on github.
		// TODO: Create a negative cache for those since they'll eat our
		// freebie quota on github.
		if r.StatusCode == http.StatusNotFound {
			return GithubUserResponse{}, false, nil
		}

		if r.StatusCode < 200 || r.StatusCode > 299 {
			return GithubUserResponse{}, false, fmt.Errorf("github returned status %d for user %q", r.StatusCode, username)
		}
		jdata, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("error reading http body: %v", err)
		}

	}
	var resp GithubUserResponse
	if err := json.Unmarshal(jdata, &resp); err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("error decoding github data: %v", err)
	}

	// Only save if we didn't read from the cache (to avoid resetting
	// the file's timestamp.
	if !ok {
		if err := cachesave(username, jdata); err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("error saving cache: %v", err)
		}
	}

	return resp, true, nil
}

// cached returns the data cached in a file (for a given username). A boolean
// return indicates whether the cache exists and is valid (in which case, data
// will contains valid data) or not.
func cached(username string) ([]byte, bool, error) {
	cfile := cachefile(username)

	fi, err := os.Stat(cfile)
	if err != nil {
		// Return no error on not exists condition (use the boolean to signal).
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, false, err
	}
	// Is this file older than maxAgeDays?
	if time.Now().After(fi.ModTime().Add(maxAgeDays * 24 * time.Hour)) {
		return nil, false, nil
	}

	data, err := ioutil.ReadFile(cfile)
	if err != nil {
		return nil, false, err
	}

	return data, true, err
}

// cachesave saves cache data for a given username, creating the required
// directory structure, if needed.
func cachesave(username string, data []byte) error {
	cfile := cachefile(username)
	dir, _ := filepath.Split(cfile)

	// Create the entire directory structure (if needed).
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	if err := ioutil.WriteFile(cfile, data, 0777); err != nil {
		return err
	}
	return nil
}

// cachefile returns the name of the cache file for a given user.
func cachefile(username string) string {
	return filepath.Join(cacheDir, username+".cache")
}
