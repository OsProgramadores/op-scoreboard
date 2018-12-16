package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"io/ioutil"
)

// Point holds the points required for a challenge.
type Point struct {
	Value int `toml:"value"`
}

// Config holds the main configuration items.
type Config struct {
	// Directory where osprogramadores/op-website-hugo is cloned.
	WebsiteDir string `toml:"website_dir"`

	// Directory where osprogramadores/op-desafios is cloned.
	ChallengesDir string `toml:"challenges_dir"`

	// Go Template directory.
	TemplateDir string `toml:"template_dir"`

	// Points per challenge. This is a map where they key is taken from the
	// challenge name (currently a number but could be anything in the future)
	// and the value is the number of points this challenge is worth.
	Points map[string]Point `toml:"points"`

	// Ignore these usernames (admins, and others that don't benefit
	// from showing in the scoreboard).
	IgnoreUsers []string `toml:"ignore_users"`
}

// parseConfig parses the configuration string from the slice of bytes
// containing the TOML config read from disk and performs basic sanity checking
// of configuration items.
func parseConfig(r io.Reader) (Config, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	if _, err := toml.Decode(string(data), &config); err != nil {
		return Config{}, err
	}
	fmt.Printf("Config is %+v\n", config)

	// Check mandatory fields.
	// if config.Username == "" || config.Password == "" || config.ClientID == "" || config.Secret == "" {
	//	return botConfig{}, errors.New("usename/password/client_id/secret cannot be null")
	//}

	return config, nil
}
