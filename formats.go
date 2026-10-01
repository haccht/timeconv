package timeconv

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	shortWeekday = `(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)`
	longWeekday  = `(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)`
	month        = `(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)`
	day          = `[ 0-3][0-9]`
	clock        = `[0-2][0-9]:[0-5][0-9]:[0-6][0-9]`
	zone         = `[A-Za-z]{3,5}`
	year         = `[0-9]{4}`
)

type formatDefinition struct {
	name            string
	layout          string
	epochUnitMicros int64
	pattern         *regexp.Regexp
}

func layoutFormat(name, layout, pattern string) *formatDefinition {
	return &formatDefinition{name: name, layout: layout, pattern: regexp.MustCompile(pattern)}
}

func epochFormat(name string, unitMicros int64, pattern string) *formatDefinition {
	return &formatDefinition{name: name, epochUnitMicros: unitMicros, pattern: regexp.MustCompile(pattern)}
}

// The order is the auto-detection priority. More precise formats precede
// formats that can match their prefix.
var formatDefinitions = []*formatDefinition{
	layoutFormat("rfc3339nano", time.RFC3339Nano, year+`-[0-1][0-9]-[0-3][0-9]T`+clock+`(?:\.[0-9]+)?(?:Z|[+-][0-2][0-9]:[0-5][0-9])`),
	layoutFormat("rfc3339", time.RFC3339, year+`-[0-1][0-9]-[0-3][0-9]T`+clock+`(?:\.[0-9]+)?(?:Z|[+-][0-2][0-9]:[0-5][0-9])`),
	layoutFormat("datetime", time.DateTime, year+`-[0-1][0-9]-[0-3][0-9] `+clock),
	layoutFormat("dateonly", time.DateOnly, year+`-[0-1][0-9]-[0-3][0-9]`),
	layoutFormat("ansic", time.ANSIC, shortWeekday+` `+month+` `+day+` `+clock+` `+year),
	layoutFormat("unixdate", time.UnixDate, shortWeekday+` `+month+` `+day+` `+clock+` `+zone+` `+year),
	layoutFormat("rubydate", time.RubyDate, shortWeekday+` `+month+` [0-3][0-9] `+clock+` [+-][0-9]{4} `+year),
	layoutFormat("rfc822z", time.RFC822Z, `[0-3][0-9] `+month+` [0-9]{2} [0-2][0-9]:[0-5][0-9] [+-][0-9]{4}`),
	layoutFormat("rfc822", time.RFC822, `[0-3][0-9] `+month+` [0-9]{2} [0-2][0-9]:[0-5][0-9] `+zone),
	layoutFormat("rfc850", time.RFC850, longWeekday+`, [0-3][0-9]-`+month+`-[0-9]{2} `+clock+` `+zone),
	layoutFormat("rfc1123z", time.RFC1123Z, shortWeekday+`, [0-3][0-9] `+month+` `+year+` `+clock+` [+-][0-9]{4}`),
	layoutFormat("rfc1123", time.RFC1123, shortWeekday+`, [0-3][0-9] `+month+` `+year+` `+clock+` `+zone),
	layoutFormat("stampnano", time.StampNano, month+` `+day+` `+clock+`\.[0-9]{9}`),
	layoutFormat("stampmicro", time.StampMicro, month+` `+day+` `+clock+`\.[0-9]{6}`),
	layoutFormat("stampmilli", time.StampMilli, month+` `+day+` `+clock+`\.[0-9]{3}`),
	layoutFormat("stamp", time.Stamp, month+` `+day+` `+clock),
	layoutFormat("timeonly", time.TimeOnly, clock),
	layoutFormat("kitchen", time.Kitchen, `[0-9]{1,2}:[0-5][0-9](?:AM|PM)`),
	epochFormat("unix-micro", 1, `[+-]?[0-9]{15,16}(?:\.[0-9]+)?`),
	epochFormat("unix-milli", 1e3, `[+-]?[0-9]{12,13}(?:\.[0-9]+)?`),
	epochFormat("unix", 1e6, `[+-]?[0-9]{9,10}(?:\.[0-9]+)?`),
}

var formatsByName = func() map[string]*formatDefinition {
	definitions := make(map[string]*formatDefinition, len(formatDefinitions))
	for _, definition := range formatDefinitions {
		definitions[definition.name] = definition
	}
	return definitions
}()

var (
	yearFirstFormats  = definitionsByName("rfc3339nano", "rfc3339", "datetime", "dateonly")
	digitFirstFormats = definitionsByName("rfc822z", "rfc822", "timeonly", "kitchen")
	textFirstFormats  = definitionsByName(
		"ansic", "unixdate", "rubydate", "rfc850", "rfc1123z", "rfc1123",
		"stampnano", "stampmicro", "stampmilli", "stamp",
	)
)

func definitionsByName(names ...string) []*formatDefinition {
	definitions := make([]*formatDefinition, len(names))
	for i, name := range names {
		definitions[i] = formatsByName[name]
	}
	return definitions
}

// Match describes a timestamp embedded in a larger string. Start and End are
// byte offsets into the original string.
type Match struct {
	Start int
	End   int
	Time  time.Time
}

// FindAllInLocation finds and parses timestamps matching a named format.
// It returns valid matches and the first invalid candidate error, if any.
func FindAllInLocation(value, format string, loc *time.Location) ([]Match, error) {
	if loc == nil {
		loc = time.UTC
	}
	definition, ok := formatsByName[strings.ToLower(format)]
	if !ok {
		return nil, fmt.Errorf("unsupported timestamp format: %s", format)
	}
	indices := definition.pattern.FindAllStringIndex(value, -1)
	matches := make([]Match, 0, len(indices))
	var firstErr error
	for _, index := range indices {
		start, end := index[0], index[1]
		if !hasTokenBoundaries(value, start, end) {
			continue
		}
		parsed, err := parseDefinition(value[start:end], definition, loc)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		matches = append(matches, Match{Start: start, End: end, Time: parsed})
	}
	return matches, firstErr
}

func detect(value string, loc *time.Location) (time.Time, error) {
	if definition := detectEpochFormat(value); definition != nil {
		return parseDefinition(value, definition, loc)
	}

	var candidates []*formatDefinition
	switch {
	case len(value) >= 5 && allDigits(value[:4]) && value[4] == '-':
		candidates = yearFirstFormats
	case len(value) > 0 && value[0] >= '0' && value[0] <= '9':
		candidates = digitFirstFormats
	case len(value) > 0 && (value[0] >= 'A' && value[0] <= 'Z' || value[0] >= 'a' && value[0] <= 'z'):
		candidates = textFirstFormats
	}

	for _, definition := range candidates {
		index := definition.pattern.FindStringIndex(value)
		if index == nil || index[0] != 0 || index[1] != len(value) {
			continue
		}
		if parsed, err := parseDefinition(value, definition, loc); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown format: %s", value)
}

func detectEpochFormat(value string) *formatDefinition {
	unsigned := strings.TrimPrefix(strings.TrimPrefix(value, "+"), "-")
	integer := unsigned
	if dot := strings.IndexByte(unsigned, '.'); dot >= 0 {
		integer = unsigned[:dot]
		fraction := unsigned[dot+1:]
		if fraction == "" || strings.ContainsRune(fraction, '.') || !allDigits(fraction) {
			return nil
		}
	}
	if !allDigits(integer) {
		return nil
	}
	switch len(integer) {
	case 9, 10:
		return formatsByName["unix"]
	case 12, 13:
		return formatsByName["unix-milli"]
	case 15, 16:
		return formatsByName["unix-micro"]
	default:
		return nil
	}
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func parseDefinition(value string, definition *formatDefinition, loc *time.Location) (time.Time, error) {
	if definition.epochUnitMicros != 0 {
		return parseEpoch(value, definition.epochUnitMicros*1e3)
	}
	return time.ParseInLocation(definition.layout, value, loc)
}

func hasTokenBoundaries(value string, start, end int) bool {
	if start > 0 {
		before, _ := utf8.DecodeLastRuneInString(value[:start])
		if isTokenRune(before) {
			return false
		}
	}
	if end < len(value) {
		after, _ := utf8.DecodeRuneInString(value[end:])
		if isTokenRune(after) {
			return false
		}
		if after == '.' && end+1 < len(value) && value[end+1] >= '0' && value[end+1] <= '9' {
			return false
		}
	}
	return true
}

func isTokenRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
