package main

import (
	"errors"
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

	switch {
	case config.WebsiteDir == "":
		return Config{}, errors.New("WebsiteDir is empty")
	case config.ChallengesDir == "":
		return Config{}, errors.New("ChallengesDir is empty")
	case config.TemplateDir == "":
		return Config{}, errors.New("TemplateDir is empty")
	}

	return config, nil
}
