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
	// Check the negative cache. This is for users who deleted their
	// accounts on github (previously got a 404). Don't try these
	// since these requests eat our freebie quota.
	_, ok, err := cached(negativeCacheFile(username), maxAgeDays*24*time.Hour)
	if err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("negative cache read error for %q: %v", username, err)
	}
	if ok {
		return GithubUserResponse{}, false, nil
	}

	// Attempt to fetch json from cache first.
	jdata, ok, err := cached(cacheFile(username), maxAgeDays*24*time.Hour)
	if err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("cache read error for %q: %v", username, err)
	}
	// Not in cache. Fetch from github.
	if !ok {
		r, err := http.Get("https://api.github.com/users/" + username)
		if err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("error retrieving github user %q: %v", username, err)
		}
		// Save username in our negative cache if we get a 404.
		if r.StatusCode == http.StatusNotFound {
			if err = cachesave(negativeCacheFile(username), []byte{}); err != nil {
				return GithubUserResponse{}, false, fmt.Errorf("negative cache write error for %q: %v", username, err)
			}
			return GithubUserResponse{}, false, nil
		}

		if r.StatusCode < 200 || r.StatusCode > 299 {
			return GithubUserResponse{}, false, fmt.Errorf("github returned status %d for user %q", r.StatusCode, username)
		}
		jdata, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("error reading http body for user %q: %v", username, err)
		}
	}
	var resp GithubUserResponse
	if err := json.Unmarshal(jdata, &resp); err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("error decoding github data: %v", err)
	}

	// Only save if we didn't read from the cache.
	// This avois prevents the unnecessary reset of the file's timestamp.
	if !ok {
		if err := cachesave(username, jdata); err != nil {
			return GithubUserResponse{}, false, fmt.Errorf("cache write error for %q: %v", username, err)
		}
	}

	return resp, true, nil
}

// cached returns the data cached in a file. A duration specifies for how long
// data in the cache is valid. Three values are returned: a slice of bytes
// containing the data in the cache file (if considered valid), a boolean
// indicating whether the data is valid or not (expired, etc), and an error.
func cached(cachefile string, exp time.Duration) ([]byte, bool, error) {
	fi, err := os.Stat(cachefile)
	if err != nil {
		// Return no error on not exists condition (use the boolean to signal).
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, false, err
	}
	// Is this file older than maxAgeDays?
	if time.Now().After(fi.ModTime().Add(exp)) {
		return nil, false, nil
	}

	data, err := ioutil.ReadFile(cachefile)
	if err != nil {
		return nil, false, err
	}

	return data, true, err
}

// cachesave saves cache data into a file, creating the required directory
// structure, if needed.
func cachesave(cachefile string, data []byte) error {
	dir, _ := filepath.Split(cachefile)

	// Create the entire directory structure (if needed).
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	if err := ioutil.WriteFile(cachefile, data, 0777); err != nil {
		return err
	}
	return nil
}

// cacheFile returns the name of the cache file for a given user.
func cacheFile(username string) string {
	return filepath.Join(cacheDir, username+".cache")
}

// negativeCachefile returns the name of the negative cache file for a given
// user.
func negativeCacheFile(username string) string {
	return filepath.Join(cacheDir, username+".negcache")
}
