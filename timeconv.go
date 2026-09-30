// Package timeconv parses and formats timestamps in common named and Go layouts.
package timeconv

import (
	"fmt"
	"math/big"
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

// Parse parses value using a named format, a Go layout, or automatic detection
// when format is empty or "auto". Timestamps without a zone are interpreted in UTC.
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
	if name == "" || name == "auto" {
		return detect(value, loc)
	}
	if definition, ok := formatsByName[name]; ok {
		return parseDefinition(value, definition, loc)
	}
	return time.ParseInLocation(format, value, loc)
}

// Format formats t using a named format or arbitrary Go layout.
func Format(t time.Time, format string) string {
	name := strings.ToLower(format)
	if definition, ok := formatsByName[name]; ok && definition.epochUnitMicros != 0 {
		value := float64(t.UnixMicro())
		return strconv.FormatFloat(value/float64(definition.epochUnitMicros), 'f', -1, 64)
	}
	if definition, ok := formatsByName[name]; ok {
		return t.Format(definition.layout)
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
