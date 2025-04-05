package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

var teamMap = map[string]string{}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: go run main.go <image-dir> <team-name>")
	}

	var err error

	imageDir := os.Args[1]
	teamName := os.Args[2]

	teamMap, err = buildTeamMapFromImages(imageDir)
	if err != nil {
		log.Fatal("Failed to build team map:", err)
	}

	s, _ := json.MarshalIndent(teamMap, "", "\t")
	fmt.Print(string(s))

	matchedTeam, err := findClosestTeamName(teamName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Matched input to team: %s\n", matchedTeam)

	var output []string
	week := 0 // global week counter

	err = filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".jpg")) {
			titles, w, err := extractMatches(path, matchedTeam, week)
			if err != nil {
				return err
			}
			output = append(output, titles...)
			week = w // update global week
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling JSON:", err)
	}

	err = os.WriteFile("titles.json", jsonOutput, 0644)
	if err != nil {
		log.Fatal("Error writing to file:", err)
	}

	fmt.Println("✅ titles.json written successfully")
}

func buildTeamMapFromImages(dir string) (map[string]string, error) {
	teamMap := make(map[string]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".jpg")) {
			client := gosseract.NewClient()
			defer client.Close()
			client.SetImage(path)

			text, err := client.Text()
			if err != nil {
				return err
			}

			// Inline logic to extract team map
			lines := strings.Split(text, "\n")
			teamEntryPattern := regexp.MustCompile(`(\d{1,2})\s+([A-Za-z ]*)\(`)

			for _, line := range lines {
				line = strings.TrimSpace(line)
				matches := teamEntryPattern.FindAllStringSubmatch(line, -1)
				for _, match := range matches {
					//fmt.Println(match)
					if len(match) == 3 {
						numStr := match[1]
						name := strings.TrimSpace(match[2])

						num, err := strconv.Atoi(numStr)
						if err != nil || num > 15 {
							continue
						}
						if _, exists := teamMap[numStr]; !exists {
							teamMap[numStr] = name

						}
					}
				}
			}
		}
		return nil
	})

	return teamMap, err
}

func findClosestTeamName(input string) (string, error) {
	lowerName := strings.ToLower(input)

	for _, name := range teamMap {
		if strings.Contains(strings.ToLower(name), lowerName) {
			return name, nil
		}
	}

	return "", fmt.Errorf("no team found")
}

func extractMatches(imagePath, teamName string, startWeek int) ([]string, int, error) {
	client := gosseract.NewClient()
	defer client.Close()
	client.SetImage(imagePath)

	text, err := client.Text()
	if err != nil {
		return nil, startWeek, err
	}

	lines := strings.Split(text, "\n")
	var currentDate string
	week := startWeek
	var titles []string

	datePattern := regexp.MustCompile(`(?i)(April|May)\s+\d+`)
	matchPattern := regexp.MustCompile(`(\d)\s*[xX]\s*(\d)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if datePattern.MatchString(line) {
			currentDate = datePattern.FindString(line)
			week++
		} else if matchPattern.MatchString(line) {
			matches := matchPattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				team1 := teamMap[match[1]]
				team2 := teamMap[match[2]]

				if team1 == teamName || team2 == teamName {
					var matchup string
					if team1 == teamName {
						matchup = fmt.Sprintf("%s vs %s", team1, team2)
					} else {
						matchup = fmt.Sprintf("%s vs %s", team2, team1)
					}
					titles = append(titles, fmt.Sprintf("%s - Week %d Game 1 - %s", formatDate(currentDate), week, matchup))
					titles = append(titles, fmt.Sprintf("%s - Week %d Game 2 - %s", formatDate(currentDate), week, matchup))
				}
			}
		}
	}

	return titles, week, nil
}

func formatDate(s string) string {
	monthMap := map[string]string{
		"April": "4",
		"May":   "5",
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return s
	}
	month := monthMap[parts[0]]
	return fmt.Sprintf("%s/%s/25", month, parts[1])
}
