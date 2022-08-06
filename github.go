// github.go - ham fisted accesses to github /user with caching.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	backoff "github.com/cenkalti/backoff/v4"
)

const (
	// Maximum number of retries on Github API.
	maxTries = 10
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

// readFromGithub reads data from a user using the github API (v3).  Returns
// the data read from Github (json), a boolean indicating whether we found a
// valid user or not, and an error.
func readFromGithub(username, token string) ([]byte, bool, error) {
	var (
		resp *http.Response
		try  int
		err  error
	)

	log.Printf("Fetching data for github user %s", username)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/users/"+username, nil)
	if err != nil {
		return nil, false, fmt.Errorf("error forming GET request for user %q: %v", username, err)
	}
	if token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	// Returning nil will cause an exit from the Retry function. The 'err' variable
	// indicates an error that needs to be handled outside the function.
	backoff.Retry(func() error {
		try++
		err = nil

		if try >= maxTries {
			err = fmt.Errorf("maximum number of retries reached (%d), user: %s", maxTries, username)
			return nil
		}

		resp, err = client.Do(req)
		if err != nil {
			m := fmt.Sprintf("error on GET for github user %q: %v (attempt %d)", username, err, try)
			log.Print(m)
			return errors.New(m)
		}

		// Return immediately if we can't find the github user. We set err to
		// nil since we don't want to abort the entire program for this.
		if resp.StatusCode == 404 {
			log.Printf("Github user not found: %s", username)
			return nil
		}

		// Retriable codes.
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			m := fmt.Sprintf("github returned status %d (%s) for user %q (attempt %d)", resp.StatusCode, resp.Status, username, try)
			log.Print(m)
			return errors.New(m)
		}
		return nil
	}, backoff.NewExponentialBackOff())

	if err != nil {
		return nil, false, err
	}

	// Indicate invalid user (but no error) if we got a 404. This is ugly.
	if resp.StatusCode == 404 {
		return nil, false, nil
	}

	jdata, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("error reading http body for user %q: %v", username, err)
	}
	return jdata, true, nil
}

// githubUserInfo returns github information about a given username.  A boolean
// flag is returned to indicate if the user was found.
func githubUserInfo(username string, token string) (GithubUserResponse, bool, error) {
	jdata, ok, err := readFromGithub(username, token)
	// Error or user not found?
	if err != nil || !ok {
		return GithubUserResponse{}, false, err
	}

	// Unmarshal the JSON and run some basic checks.
	var resp GithubUserResponse
	if err := json.Unmarshal(jdata, &resp); err != nil {
		return GithubUserResponse{}, false, fmt.Errorf("error decoding github data: %v", err)
	}
	if resp.Login == "" {
		return GithubUserResponse{}, false, fmt.Errorf("got bad json from github: %s", string(jdata))
	}

	return resp, true, nil
}
