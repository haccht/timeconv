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

const helpText = `
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

  Arbitrary formats are also supported. See https://pkg.go.dev/time as a reference.`

var layouts = map[string]string{
	"ansic": time.ANSIC,
	"unixdate": time.UnixDate,
	"rubydate": time.RubyDate,
	"rfc822": time.RFC822,
	"rfc822z": time.RFC822Z,
	"rfc850": time.RFC850,
	"rfc1123": time.RFC1123,
	"rfc1123z": time.RFC1123Z,
	"rfc3339": time.RFC3339,
	"rfc3339nano": time.RFC3339Nano,
	"kitchen": time.Kitchen,
	"stamp": time.Stamp,
	"stampmilli": time.StampMilli,
	"stampmicro": time.StampMicro,
	"stampnano": time.StampNano,
	"datetime": time.DateTime,
	"dateonly": time.DateOnly,
	"timeonly": time.TimeOnly,
}

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

func parseValue(v, format string) (time.Time, error) {
	layout, ok := layouts[format]
	if ok {
		return time.Parse(layout, v)
	}

	switch layout {
	case "unix":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", v)
		}
		return time.UnixMicro(int64(f * 1000000)), nil
	case "unix.milli":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", v)
		}
		return time.UnixMicro(int64(f * 1000)), nil
	case "unix.micro":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", v)
		}
		return time.UnixMicro(int64(f)), nil
	default:
		return time.Time{}, fmt.Errorf("failed to parse time: %s", v)
	}
}

func formatValue(t time.Time, format string) string {
	layout, ok := layouts[format]
	if ok {
		return t.Format(layout)
	}

	switch layout {
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
		fmt.Fprint(&message, helpText)

		fmt.Println(message.String())
		os.Exit(0)
	}

	opts.In = strings.ToLower(opts.In)
	opts.Out = strings.ToLower(opts.Out)

	loc := time.Local
	if opts.Tz != "" {
		l, err := time.LoadLocation(opts.Tz)
		if err != nil {
			return err
		}

		loc = l
	}

	if opts.Now {
		t := time.Now()

		t = t.Add(parseDuration(opts.Add))
		t = t.Add(parseDuration(opts.Sub) * -1)
		t = t.In(loc)
		fmt.Println(formatValue(t, opts.Out))
	} else {
		scanner := getScanner(args)
		w := bufio.NewWriter(os.Stdout)
		defer w.Flush()

		for scanner.Scan() {
			v := strings.TrimSpace(scanner.Text())
			t, err := parseValue(v, opts.In)
			if err != nil {
				return err
			}

			t = t.Add(parseDuration(opts.Add))
			t = t.Add(parseDuration(opts.Sub) * -1)
			t = t.In(loc)
			fmt.Println(formatValue(t, opts.Out))
		}

		if err := scanner.Err(); err != nil {
			return scanner.Err()
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
