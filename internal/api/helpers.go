package api

import (
	"regexp"
	"strings"
)

var headerRe = regexp.MustCompile(`\[(\w+)\s+"([^"]+)"\]`)

func parsePGNHeaders(pgn string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(pgn, "\n") {
		if !strings.HasPrefix(line, "[") {
			continue
		}
		m := headerRe.FindStringSubmatch(line)
		if len(m) == 3 {
			out[m[1]] = m[2]
		}
	}
	return out
}

var gameIDRe = regexp.MustCompile(`.*/game/[^/]+/([0-9]+)`)

func extractGameID(url string) string {
	m := gameIDRe.FindStringSubmatch(url)
	if len(m) == 2 {
		return m[1]
	}
	return url
}

