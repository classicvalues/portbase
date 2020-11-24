package osdetail

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	segmentsSplitter = regexp.MustCompile("[^A-Za-z0-9]*[A-Z]?[a-z0-9]*")
	nameOnly         = regexp.MustCompile("^[A-Za-z0-9]+$")
	delimiters       = regexp.MustCompile("^[^A-Za-z0-9]+")
)

// GenerateBinaryNameFromPath generates a more human readable binary name from
// the given path. This function is used as fallback in the GetBinaryName
// functions.
func GenerateBinaryNameFromPath(path string) string {
	// Get file name from path.
	_, fileName := filepath.Split(path)

	// Split up into segments.
	segments := segmentsSplitter.FindAllString(fileName, -1)

	// Remove last segment if it's an extension.
	if len(segments) >= 2 &&
		strings.HasPrefix(segments[len(segments)-1], ".") {
		segments = segments[:len(segments)-1]
	}

	// Go through segments and collect name parts.
	nameParts := make([]string, 0, len(segments))
	var fragments string
	for _, segment := range segments {
		// Group very short segments.
		if len(segment) <= 3 {
			fragments += segment
			continue
		} else if fragments != "" {
			nameParts = append(nameParts, fragments)
			fragments = ""
		}

		// Add segment to name.
		nameParts = append(nameParts, segment)
	}
	// Add last fragment.
	if fragments != "" {
		nameParts = append(nameParts, fragments)
	}

	// Post-process name parts
	for i := range nameParts {
		// Remove any leading delimiters.
		nameParts[i] = delimiters.ReplaceAllString(nameParts[i], "")

		// Title-case name-only parts.
		if nameOnly.MatchString(nameParts[i]) {
			nameParts[i] = strings.Title(nameParts[i])
		}
	}

	return strings.Join(nameParts, " ")
}

func cleanFileDescription(fields []string) string {
	// If there is a 1 or 2 character delimiter field, only use fields before it.
	endIndex := len(fields)
	for i, field := range fields {
		// Ignore the first field as well as fields with more than two characters.
		if i >= 1 && len(field) <= 2 && !nameOnly.MatchString(field) {
			endIndex = i
			break
		}
	}

	// Concatenate name
	binName := strings.Join(fields[:endIndex], " ")

	// If there are multiple sentences, only use the first.
	if strings.Contains(binName, ". ") {
		binName = strings.SplitN(binName, ". ", 2)[0]
	}

	return binName
}