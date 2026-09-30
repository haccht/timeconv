// Package timeconv parses and formats timestamps in common named and Go layouts.
package timeconv

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LayoutExamples describes the named formats accepted by Parse and Format.
const LayoutExamples = `  ANSIC       "Mon Jan _2 15:04:05 2006"
  UnixDate    "Mon Jan _2 15:04:05 MST 2006"
  RubyDate    "Mon Jan 02 15:04:05 -0700 2006"
  RFC822      "02 Jan 06 15:04 MST"
  RFC822Z     "02 Jan 06 15:04 -0700"
  RFC850      "Monday, 02-Jan-06 15:04:05 MST"
  RFC1123     "Mon, 02 Jan 2006 15:04:05 MST"
  RFC1123Z    "Mon, 02 Jan 2006 15:04:05 -0700"
  RFC3339     "2006-01-02T15:04:05Z07:00"
  RFC3339Nano "2006-01-02T15:04:05.999999999Z07:00"
  Kitchen     "3:04PM"
  Stamp       "Jan _2 15:04:05"
  StampMilli  "Jan _2 15:04:05.000"
  StampMicro  "Jan _2 15:04:05.000000"
  StampNano   "Jan _2 15:04:05.000000000"
  DateTime    "2006-01-02 15:04:05"
  DateOnly    "2006-01-02"
  TimeOnly    "15:04:05"
  Unix        "1136239445"
  Unix-Milli  "1136239445000"
  Unix-Micro  "1136239445000000"

  Arbitrary formats are also supported. See https://pkg.go.dev/time as a reference.`

var knownLayouts = map[string]string{
	"ansic": time.ANSIC, "unixdate": time.UnixDate, "rubydate": time.RubyDate,
	"rfc822": time.RFC822, "rfc822z": time.RFC822Z, "rfc850": time.RFC850,
	"rfc1123": time.RFC1123, "rfc1123z": time.RFC1123Z, "rfc3339": time.RFC3339,
	"rfc3339nano": time.RFC3339Nano, "kitchen": time.Kitchen, "stamp": time.Stamp,
	"stampmilli": time.StampMilli, "stampmicro": time.StampMicro, "stampnano": time.StampNano,
	"datetime": time.DateTime, "dateonly": time.DateOnly, "timeonly": time.TimeOnly,
}

var epochLayouts = map[string]int64{"unix": 1e6, "unix-milli": 1e3, "unix-micro": 1}

type guessRule struct {
	re      *regexp.Regexp
	layouts []string
}

var guessRules = []guessRule{
	{regexp.MustCompile(`^\d{4}`), []string{"rfc3339", "rfc3339nano", "datetime", "dateonly"}},
	{regexp.MustCompile(`[A-Za-z]{3,4}|[+-]\d{4}`), []string{"unixdate", "rubydate", "rfc822", "rfc822z", "rfc850", "rfc1123", "rfc1123z", "rfc3339", "rfc3339nano"}},
	{regexp.MustCompile(`^[A-Za-z]{3},?`), []string{"ansic", "unixdate", "rubydate", "rfc822", "rfc822z", "rfc850", "rfc1123", "rfc1123z", "stamp", "stampmilli", "stampmicro", "stampnano"}},
	{regexp.MustCompile(`\d{2}:\d{2}:\d{2}`), []string{"datetime", "timeonly", "ansic", "unixdate", "rubydate", "rfc850", "rfc1123", "rfc1123z"}},
	{regexp.MustCompile(`\d{1,2}:\d{2}(AM|PM)`), []string{"kitchen"}},
}

// Parse parses value using a named format, a Go layout, or automatic detection
// when format is empty. Timestamps without a zone are interpreted in UTC.
func Parse(value, format string) (time.Time, error) {
	return ParseInLocation(value, format, time.UTC)
}

// ParseInLocation is like Parse, but interprets timestamps without an explicit
// zone in loc. Explicit offsets and zones in value take precedence.
func ParseInLocation(value, format string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}
	name := strings.ToLower(format)
	if name == "" {
		return guess(value, loc)
	}
	if unitMicros, ok := epochLayouts[name]; ok {
		return parseEpoch(value, unitMicros*1e3)
	}
	if layout, ok := knownLayouts[name]; ok {
		return time.ParseInLocation(layout, value, loc)
	}
	return time.ParseInLocation(format, value, loc)
}

// Format formats t using a named format or arbitrary Go layout.
func Format(t time.Time, format string) string {
	name := strings.ToLower(format)
	if scale, ok := epochLayouts[name]; ok {
		value := float64(t.UnixMicro())
		return strconv.FormatFloat(value/float64(scale), 'f', -1, 64)
	}
	if layout, ok := knownLayouts[name]; ok {
		return t.Format(layout)
	}
	return t.Format(format)
}

func parseEpoch(value string, unitNanos int64) (time.Time, error) {
	rational, ok := new(big.Rat).SetString(value)
	if !ok {
		return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", value)
	}
	rational.Mul(rational, new(big.Rat).SetInt64(unitNanos))
	totalNanos := new(big.Int).Quo(rational.Num(), rational.Denom())
	seconds, nanos := new(big.Int), new(big.Int)
	seconds.QuoRem(totalNanos, big.NewInt(int64(time.Second)), nanos)
	if !seconds.IsInt64() || !nanos.IsInt64() {
		return time.Time{}, fmt.Errorf("epoch time out of range: %s", value)
	}
	return time.Unix(seconds.Int64(), nanos.Int64()), nil
}

func guess(value string, loc *time.Location) (time.Time, error) {
	if format := guessEpochFormat(value); format != "" {
		return ParseInLocation(value, format, loc)
	}
	for _, rule := range guessRules {
		if !rule.re.MatchString(value) {
			continue
		}
		for _, layout := range rule.layouts {
			if parsed, err := ParseInLocation(value, layout, loc); err == nil {
				return parsed, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("unknown format: %s", value)
}

func guessEpochFormat(value string) string {
	integer := strings.TrimPrefix(strings.TrimPrefix(value, "+"), "-")
	if dot := strings.IndexByte(integer, '.'); dot >= 0 {
		integer = integer[:dot]
	}
	if integer == "" || strings.IndexFunc(integer, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return ""
	}
	switch len(integer) {
	case 9, 10:
		return "unix"
	case 12, 13:
		return "unix-milli"
	case 15, 16:
		return "unix-micro"
	default:
		return ""
	}
}
