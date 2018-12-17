package main

import (
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// Glob matching challenges directory.
	// Common syntax is <anything>-<challenge_name_or_number>/author
	challengesGlob = "./desafio-*/*"
)

// playerChallenge holds one user/challenge pair read from the disk.
type playerChallenge struct {
	username  string
	challenge string
}

// playerScore holds the total number of points and completed challenges for
// one particular player.
type playerScore struct {
	points    int
	completed []string
}

// scoreboardEntry holds one entry in the scoreboard. It contains all
// information required to emit output for this player.
type scoreboardEntry struct {
	rank       int
	fullName   string
	githubUser string
	avatarURL  string
	score      playerScore
	// True if this user is the last in a group. Typically the last of a number
	// of people with the same score.
	lastInGroup bool
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

	challenges, err := readChallenges(config.ChallengesDir)
	if err != nil {
		log.Fatal(err)
	}

	scores, err := makePlayerScores(challenges, config.IgnoreUsers, config.Points)
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(createScoreboard(scores))
}

// readChallenges reads all relevant directories under ddir and
// return a list containing the users and challenges found.
func readChallenges(ddir string) ([]playerChallenge, error) {
	var ret []playerChallenge

	dpaths, err := filepath.Glob(filepath.Join(ddir, challengesGlob))
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

// makePlayerScores generates a map of playerScores structures from the list of
// player/challenges keyed on github username. Any username on the 'ignore'
// list will be silently ignored. Uses the pointsConfig map to calculate how
// much each challenge is worth in points.
func makePlayerScores(challenges []playerChallenge, ignore []string, pointsConfig map[string]Point) (map[string]playerScore, error) {
	scores := map[string]playerScore{}

	for _, c := range challenges {
		// Make sure user is not ignored.
		if inSlice(ignore, c.username) {
			continue
		}

		// Compute score for this player/challenge
		pts, err := calcScores(c, pointsConfig)
		if err != nil {
			return nil, err
		}
		s, ok := scores[c.username]
		if !ok {
			s = playerScore{}
		}
		s.points += pts

		// Add challenge to list of completed for this player
		if !inSlice(s.completed, c.challenge) {
			s.completed = append(s.completed, c.challenge)
		}
		scores[c.username] = s
	}
	return scores, nil
}

// parsePath parses a path under challengesDir and returns the user and
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

func createScoreboard(scores map[string]playerScore) []scoreboardEntry {
	var scoreboard []scoreboardEntry

	for u, s := range scores {
		sbe := scoreboardEntry{
			fullName:   "not implemented",
			githubUser: u,
			avatarURL:  "https://upload.wikimedia.org/wikipedia/en/8/8b/Avatar_2_logo.jpg?1544987538381",
			score:      s,
		}
		scoreboard = append(scoreboard, sbe)
	}

	// Reverse sort by points.
	sort.Slice(scoreboard, func(i, j int) bool {
		return scoreboard[i].score.points > scoreboard[j].score.points
	})

	// Scan scoreboard and add rank and end of group indicators.
	rank := 0
	oldpoints := 0

	for k := range scoreboard {
		points := scoreboard[k].score.points
		if points != oldpoints {
			rank++
			if k != 0 {
				scoreboard[k-1].lastInGroup = true
			}
		}
		scoreboard[k].rank = rank
		oldpoints = points
	}
	// Last element is always marked as last in group.
	scoreboard[len(scoreboard)-1].lastInGroup = true

	return scoreboard
}
