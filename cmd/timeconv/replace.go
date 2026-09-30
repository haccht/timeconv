package main

import (
	"strings"

	"github.com/haccht/timeconv"
)

func replaceNamedTimestamps(line string, opts *options) (string, error) {
	matches, firstErr := timeconv.FindAllInLocation(line, opts.replace, opts.inLoc.Location)
	if len(matches) == 0 {
		return line, firstErr
	}

	var result strings.Builder
	result.Grow(len(line))
	last := 0
	for _, match := range matches {
		result.WriteString(line[last:match.Start])
		parsed := modifyTime(match.Time, opts.loc, opts.add, opts.sub)
		result.WriteString(timeconv.Format(parsed, opts.out))
		last = match.End
	}
	result.WriteString(line[last:])
	return result.String(), firstErr
}
