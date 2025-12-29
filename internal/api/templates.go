package api

import (
	"encoding/json"
	"html/template"
	"net/url"
	"path/filepath"
)

func LoadTemplates() (*template.Template, error) {
	funcs := template.FuncMap{
		// slice creates a slice from variadic string arguments
		"slice": func(args ...string) []string {
			return args
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		// seq returns a sequence of integers from start to end inclusive.
		"seq": func(start, end int) []int {
			if end < start {
				return []int{}
			}
			nums := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				nums = append(nums, i)
			}
			return nums
		},
		// getFilter extracts a single value from url.Values (returns first value or empty string)
		"getFilter": func(values url.Values, key string) string {
			if values == nil {
				return ""
			}
			return values.Get(key)
		},
		// urlquery URL-encodes a string
		"urlquery": func(s string) string {
			return url.QueryEscape(s)
		},
		// json marshals a value to JSON string
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}

	t := template.New("base").Funcs(funcs)

	patterns := []string{
		"web/templates/layouts/*.html",
		"web/templates/pages/*.html",
		"web/templates/partials/*.html",
	}
	for _, p := range patterns {
		if matches, _ := filepath.Glob(p); len(matches) == 0 {
			continue
		}
		if _, err := t.ParseGlob(p); err != nil {
			return nil, err
		}
	}

	return t, nil
}
