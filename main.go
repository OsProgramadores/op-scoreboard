package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Glob matching desafios.
	desafiosGlob = "./desafio-*/*"
)

// player holds all data from the contestants.
type playerInfo struct {
	fullName   string
	githubUser string
	avatarURL  string
	points     int
	challenges []string
}

// playerChallenge holds one user/challenge pair read from the disk.
type playerChallenge struct {
	username  string
	challenge string
}

// playerScore holds scores for one particular player.
type playerScore struct {
	score     int
	completed []string
}

func main() {
	//log.SetFlags(0)
	configFile := flag.String("config", "", "configuration file")
	flag.Parse()

	r, err := os.Open(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	config, err := parseConfig(r)
	if err != nil {
		log.Fatal(err)
	}

	challenges, err := readChallenges(config.DesafiosDir)
	if err != nil {
		log.Fatal(err)
	}

	scores := map[string]playerScore{}

	for _, c := range challenges {
		// Make sure user is not ignored.
		if inSlice(config.IgnoreUsers, c.username) {
			continue
		}

		// Compute score for this player/challenge
		pts, err := calcScores(c, config.Points)
		if err != nil {
			log.Fatal(err)
		}
		s, ok := scores[c.username]
		if !ok {
			s = playerScore{score: 0}
		}
		s.score += pts

		// Add challenge to list of completed for this player
		if !inSlice(s.completed, c.challenge) {
			s.completed = append(s.completed, c.challenge)
		}
		scores[c.username] = s
	}

	fmt.Printf("%+v\n", scores)
}

// readChallenges reads all relevant directories under ddir and
// return a list containing the users and challenges found.
func readChallenges(ddir string) ([]playerChallenge, error) {
	var ret []playerChallenge

	dpaths, err := filepath.Glob(filepath.Join(ddir, desafiosGlob))
	if err != nil {
		return nil, err
	}

	for _, v := range dpaths {
		username, challenge, err := parsePath(v)
		if err != nil {
			return nil, err
		}
		ret = append(ret, playerChallenge{username: username, challenge: challenge})
	}
	return ret, nil
}

// parsePath parses a path under DesafiosDir and returns the user and
// designation of that particular challenge (or error). This function assumes
// that directories under path are laid out as desafio-NN/username
func parsePath(path string) (string, string, error) {
	elems := strings.Split(path, "/")
	if len(elems) < 2 {
		return "", "", fmt.Errorf("invalid file/dir: %q", path)
	}

	username := elems[len(elems)-1]
	dpath := elems[len(elems)-2]

	// Split desafio-NN
	d := strings.Split(dpath, "-")
	if len(d) != 2 {
		return "", "", fmt.Errorf("invalid directory format: %q", dpath)
	}
	return username, d[1], nil
}

// calcScores returns the calcScores for a single username
func calcScores(challenge playerChallenge, points map[string]Point) (int, error) {
	pointvalue, ok := points[challenge.challenge]
	if !ok {
		return 0, fmt.Errorf("missing points configuration for: %q", challenge.challenge)
	}
	return pointvalue.Value, nil
}

func inSlice(sl []string, str string) bool {
	for _, v := range sl {
		if str == v {
			return true
		}
	}
	return false
}
