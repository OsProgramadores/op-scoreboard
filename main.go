package main

import (
	"flag"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"html/template"
	"io"
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

	// Name of the main template file.
	templateFile = "scoreboard.template"
)

// playerChallenge holds one user/challenge pair read from the disk.
type playerChallenge struct {
	username  string
	challenge string
}

// CompletedChallenge holds information about the challenges completed by
// a user.
type CompletedChallenge struct {
	Name   string
	Points int
}

// playerScore holds the total number of points and completed challenges for
// one particular player.
type playerScore struct {
	Points    int
	Completed []CompletedChallenge
}

// scoreboardEntry holds one entry in the scoreboard. It contains all
// information required to emit output for this player.
type scoreboardEntry struct {
	Rank       int
	GithubUser string
	Score      playerScore
	Completed  []CompletedChallenge
	// Full info from github
	GithubInfo GithubUserResponse
	// True if this is the first user in a group.
	FirstInGroup bool
	// True if this user is the last in a group. Typically the last of a number
	// of people with the same score.
	LastInGroup bool
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
	if len(challenges) == 0 {
		log.Fatal("No challenges found. Check the value of challenge_dir in the config file.")
	}

	scores, err := makePlayerScores(challenges, config.IgnoreUsers, config.Points)
	if err != nil {
		log.Fatal(err)
	}

	scoreboard, err := createScoreboard(scores)
	if err != nil {
		log.Fatal(err)
	}

	tfile := filepath.Join(config.TemplateDir, templateFile)
	if err := writeTemplateFile(filepath.Join(config.WebsiteDir, "/content/scores.md"), scoreboard, tfile); err != nil {
		log.Fatal(err)
	}
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

		// Add challenge to list of completed for this player
		if !alreadyCompleted(s.Completed, c.challenge) {
			cc := CompletedChallenge{
				Name:   c.challenge,
				Points: pts,
			}
			s.Completed = append(s.Completed, cc)
		}
		// Add total points.
		s.Points += pts

		scores[c.username] = s
	}
	return scores, nil
}

// parsePath parses a path under challengesDir and returns the user and
// designation of that particular challenge (or error). This function assumes
// that directories under path are laid out as challenge_name/username
func parsePath(path string) (string, string, error) {
	elems := strings.Split(path, "/")
	if len(elems) < 2 {
		return "", "", fmt.Errorf("invalid file/dir: %q", path)
	}

	cname := elems[len(elems)-2]
	username := elems[len(elems)-1]
	return username, cname, nil
}

// calcScores returns the calcScores for a single username
func calcScores(challenge playerChallenge, points map[string]Point) (int, error) {
	pointvalue, ok := points[challenge.challenge]
	if !ok {
		return 0, fmt.Errorf("missing points configuration for: %q", challenge.challenge)
	}
	return pointvalue.Value, nil
}

// inSlice returns true if a given string is inside a slice of strings.
func inSlice(sl []string, str string) bool {
	for _, v := range sl {
		if str == v {
			return true
		}
	}
	return false
}

// alreadyCompleted returns true if a given challenge is already in a slice of
// completeChallenge structs.
func alreadyCompleted(cc []CompletedChallenge, name string) bool {
	for _, v := range cc {
		if name == v.Name {
			return true
		}
	}
	return false
}

// createScoreboard creates a "scoreboard" slice, ready to be rendered by
// templates.  We need a slice here to make it easier to sort by points.
func createScoreboard(scores map[string]playerScore) ([]scoreboardEntry, error) {
	var scoreboard []scoreboardEntry

	for u, s := range scores {
		githubInfo, ok, err := githubUserInfo(u)
		if err != nil {
			return nil, err
		}
		// No user on github?
		if !ok {
			continue
		}

		sbe := scoreboardEntry{
			GithubUser: u,
			Score:      s,
			Completed:  s.Completed,
			GithubInfo: githubInfo,
		}

		scoreboard = append(scoreboard, sbe)
	}

	// Descending sort by points, ascending sort by username for users
	// with the same number of points.
	sort.Slice(scoreboard, func(i, j int) bool {
		if scoreboard[i].Score.Points == scoreboard[j].Score.Points {
			return strings.ToLower(scoreboard[i].GithubUser) < strings.ToLower(scoreboard[j].GithubUser)
		}
		return scoreboard[i].Score.Points > scoreboard[j].Score.Points
	})

	// Scan scoreboard and add rank and end of group indicators.
	rank := 0
	oldpoints := 0

	for k := range scoreboard {
		points := scoreboard[k].Score.Points
		if points != oldpoints {
			rank++
			scoreboard[k].FirstInGroup = true
			if k != 0 {
				scoreboard[k-1].LastInGroup = true
			}
		}
		scoreboard[k].Rank = rank
		oldpoints = points
	}
	// Last element is always marked as last in group.
	if len(scoreboard) != 0 {
		scoreboard[len(scoreboard)-1].LastInGroup = true
	}

	return scoreboard, nil
}

// writeTemplateFile writes a scoreboard to the default output file using a
// specified template file.
func writeTemplateFile(outfile string, scoreboard []scoreboardEntry, tfile string) error {
	w, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer w.Close()
	return writeTemplate(w, scoreboard, tfile)
}

// writeTemplate writes a scoreboard to an io.Writer using a specified template
// file.
func writeTemplate(w io.Writer, scoreboard []scoreboardEntry, tfile string) error {
	_, tbasefile := filepath.Split(tfile)

	t := template.New(tbasefile)
	t, err := t.ParseFiles(tfile)
	if err != nil {
		return fmt.Errorf("writeTemplate: error parsing template: %v", err)
	}
	if err = t.Execute(w, scoreboard); err != nil {
		return fmt.Errorf("writeTemplate: error executing template: %v", err)
	}
	return nil
}
