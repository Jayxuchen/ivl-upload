package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/otiai10/gosseract/v2"
)

var teamMap = map[string]string{
	"1": "Set For Life",
	"2": "Foundry",
	"3": "Bellamy and Buds",
	"4": "Pengyou Power",
	"5": "Guintu Force",
	"6": "Reclub Most Wanted",
	"7": "How I Set Your Mother",
	"8": "Kicking and Screaming",
	"9": "Haikrew",
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: go run main.go <image-dir> <team-name>")
	}

	imageDir := os.Args[1]
	teamName := os.Args[2]
	var output []string
	week := 0 // global week counter

	err := filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".png") || strings.HasSuffix(info.Name(), ".jpg")) {
			titles, w, err := extractMatches(path, teamName, week)
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
