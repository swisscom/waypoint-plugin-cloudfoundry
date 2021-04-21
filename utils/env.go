package utils

import (
	"regexp"
	"strings"
)

func ParseEnv(s string) map[string]string {
	env := map[string]string{}
	envRegex := regexp.MustCompile("^([^#].*?)=(.*)$")
	for _, line := range strings.Split(s, "\n") {
		if !envRegex.MatchString(line) {
			continue
		}

		matches := envRegex.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		env[matches[1]] = matches[2]

	}
	return env
}
