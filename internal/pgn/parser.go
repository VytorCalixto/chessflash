package pgn

import (
	"regexp"
	"strings"
)

var headerRe = regexp.MustCompile(`\[(\w+)\s+"([^"]+)"\]`)

// ParsePGNHeaders extracts PGN header tags into a map
func ParsePGNHeaders(pgn string) map[string]string {
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

// ExtractGameID extracts the game ID from a chess.com game URL
func ExtractGameID(url string) string {
	m := gameIDRe.FindStringSubmatch(url)
	if len(m) == 2 {
		return m[1]
	}
	return url
}
