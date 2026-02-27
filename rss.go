package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type RSS struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

var feedCandidates = []string{
	"public/index.xml",
	"public/feed.xml",
	"public/rss.xml",
}

var devHosts = []string{"localhost", "127.0.0.1", "0.0.0.0"}

func discoverAndParseFeed() (*RSS, error) {
	path, err := findFeedPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading feed %s: %w", path, err)
	}

	var feed RSS
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing feed %s: %w", path, err)
	}

	return &feed, nil
}

func findFeedPath() (string, error) {
	for _, candidate := range feedCandidates {
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no RSS feed found; checked %s — run `hugo` to build first", strings.Join(feedCandidates, ", "))
}

func validateProductionURL(item *RSSItem) error {
	link := strings.ToLower(item.Link)
	for _, host := range devHosts {
		if strings.Contains(link, host) {
			return fmt.Errorf("RSS feed links point to a development host — run `hugo` with your production baseURL before posting")
		}
	}
	return nil
}
