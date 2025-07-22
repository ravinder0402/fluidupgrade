package pattern

import (
	"regexp"
)

func IsBasicName(str string) bool {
	size := len(str)
	if size < 3 || size > 64 {
		return false
	}

	matched, _ := regexp.MatchString(basicNameRegex, str)

	return matched
}
