package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
)

type options struct {
	In   string `short:"i" long:"in" description:"Specify input time format" default:"RFC3339"`
	Out  string `short:"o" long:"out" description:"Specify output time format" default:"RFC3339"`
	Now  bool   `short:"n" long:"now" description:"Load currnet time as input"`
	Add  string `short:"a" long:"add" description:"Append specified duration (ex. 5m, 1.5h, 1h30m)"`
	Sub  string `short:"s" long:"sub" description:"Substract specified duration (ex. 5m, 1.5h, 1h30m)"`
	Tz   string `short:"z" long:"tz" description:"Override timezone"`
	Help bool   `short:"h" long:"help" description:"Show this help message"`
}

func getScanner(args []string) *bufio.Scanner {
	if len(args) > 0 {
		reader := strings.NewReader(strings.Join(args, "\n"))
		return bufio.NewScanner(reader)
	}
	return bufio.NewScanner(os.Stdin)
}

func parseValue(format, value string) (time.Time, error) {
	layout := loadLayout(format)

	switch strings.ToLower(layout) {
	case "unix":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", value)
		}
		return time.UnixMicro(int64(f * 1000000)), nil
	case "unix.milli":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", value)
		}
		return time.UnixMicro(int64(f * 1000)), nil
	case "unix.micro":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", value)
		}
		return time.UnixMicro(int64(f)), nil
	default:
		return time.Parse(layout, value)
	}
}

func formatValue(format string, t time.Time) string {
	layout := loadLayout(format)

	switch strings.ToLower(layout) {
	case "unix":
		f := float64(t.UnixNano()) / 1000000000
		return strconv.FormatFloat(f, 'f', -1, 64)
	case "unix.milli":
		f := float64(t.UnixNano()) / 1000000
		return strconv.FormatFloat(f, 'f', -1, 64)
	case "unix.micro":
		f := float64(t.UnixNano()) / 1000
		return strconv.FormatFloat(f, 'f', -1, 64)
	default:
		return t.Format(layout)
	}
}

func loadLayout(format string) string {
	switch strings.ToLower(format) {
	case "ansic":
		return time.ANSIC
	case "unixdate":
		return time.UnixDate
	case "rubydate":
		return time.RubyDate
	case "rfc822":
		return time.RFC822
	case "rfc822z":
		return time.RFC822Z
	case "rfc850":
		return time.RFC850
	case "rfc1123":
		return time.RFC1123
	case "rfc1123z":
		return time.RFC1123Z
	case "rfc3339":
		return time.RFC3339
	case "rfc3339nano":
		return time.RFC3339Nano
	case "kitchen":
		return time.Kitchen
	case "stamp":
		return time.Stamp
	case "stampmilli":
		return time.StampMilli
	case "stampmicro":
		return time.StampMicro
	case "stampnano":
		return time.StampNano
	case "datetime":
		return time.DateTime
	case "dateonly":
		return time.DateOnly
	case "timeonly":
		return time.TimeOnly
	default:
		return format
	}
}

func parseDuration(d string) time.Duration {
	u, err := time.ParseDuration(d)
	if err != nil {
		return 0
	}

	return u
}

func run() error {
	var opts options

	parser := flags.NewParser(&opts, flags.Default&^flags.HelpFlag)
	parser.Usage = "[Options]"

	args, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	if opts.Help {
		var message bytes.Buffer

		parser.WriteHelp(&message)
		fmt.Fprint(&message, `
Format Examples:
  ANSIC       "Mon Jan _2 15:04:05 2006"
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

  Arbitrary formats are also supported. See https://pkg.go.dev/time as a reference.`)

		fmt.Println(message.String())
		os.Exit(0)
	}

	loc := time.Local
	if opts.Tz != "" {
		l, err := time.LoadLocation(opts.Tz)
		if err != nil {
			return err
		}

		loc = l
	}

	times := []time.Time{}
	if opts.Now {
		t := time.Now()
		times = append(times, t)
	} else {
		scanner := getScanner(args)
		for scanner.Scan() {
			v := strings.TrimSpace(scanner.Text())
			t, err := parseValue(opts.In, v)
			if err == nil {
				times = append(times, t)
			}
		}

		if err := scanner.Err(); err != nil {
			return scanner.Err()
		}
	}

	for _, t := range times {
		t = t.In(loc)
		t = t.Add(parseDuration(opts.Add))
		t = t.Add(parseDuration(opts.Sub) * -1)

		fmt.Println(formatValue(opts.Out, t))
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
