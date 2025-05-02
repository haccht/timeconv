package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
)

const helpText = `Format Examples:
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
	"ansic":       time.ANSIC,
	"unixdate":    time.UnixDate,
	"rubydate":    time.RubyDate,
	"rfc822":      time.RFC822,
	"rfc822z":     time.RFC822Z,
	"rfc850":      time.RFC850,
	"rfc1123":     time.RFC1123,
	"rfc1123z":    time.RFC1123Z,
	"rfc3339":     time.RFC3339,
	"rfc3339nano": time.RFC3339Nano,
	"kitchen":     time.Kitchen,
	"stamp":       time.Stamp,
	"stampmilli":  time.StampMilli,
	"stampmicro":  time.StampMicro,
	"stampnano":   time.StampNano,
	"datetime":    time.DateTime,
	"dateonly":    time.DateOnly,
	"timeonly":    time.TimeOnly,
}

type options struct {
	In      string    `short:"i" long:"in" description:"Specify input time format (default: guess format)"`
	Out     string    `short:"o" long:"out" description:"Specify output time format" default:"rfc3339"`
	Now     bool      `short:"n" long:"now" description:"Load currnet time as input"`
	Add     string    `short:"a" long:"add" description:"Append specified duration (ex. 5m, 1.5h, 1h30m)"`
	Sub     string    `short:"s" long:"sub" description:"Substract specified duration (ex. 5m, 1.5h, 1h30m)"`
	Loc      string `short:"z" long:"loc" description:"Override timezone"`
	Pattern string   `short:"g" long:"grep" description:"Replace strings that match the regular expression"`
	Help    bool      `short:"h" long:"help" description:"Show this help message"`
}

type guessRule struct {
	re      *regexp.Regexp
	layouts []string
}

var guessRules = []guessRule{
	{regexp.MustCompile(`^\d{10,19}(?:\.\d+)?$`), []string{"unix", "unix-milli", "unix-micro"}},
	{regexp.MustCompile(`^\d{4}`), []string{"rfc3339", "rfc3339nano", "datetime", "dateonly"}},
	{regexp.MustCompile(`[A-Za-z]{3,4}|[+-]\d{4}`), []string{"unixdate", "rubydate", "rfc822", "rfc822z", "rfc850", "rfc1123", "rfc1123z", "rfc3339", "rfc3339nano"}},
	{regexp.MustCompile(`^[A-Za-z]{3},?`), []string{"ansic", "unixdate", "rubydate", "rfc822", "rfc822z", "rfc850", "rfc1123", "rfc1123z", "stamp", "stampmilli", "stampmicro", "stampnano"}},
	{regexp.MustCompile(`\d{2}:\d{2}:\d{2}`), []string{"datetime", "timeonly", "ansic", "unixdate", "rubydate", "rfc850", "rfc1123", "rfc1123z"}},
	{regexp.MustCompile(`\d{1,2}:\d{2}(AM|PM)`), []string{"kitchen"}},
}

func genScanner(args []string) *bufio.Scanner {
	if len(args) > 0 {
		reader := strings.NewReader(strings.Join(args, "\n"))
		return bufio.NewScanner(reader)
	}
	return bufio.NewScanner(os.Stdin)
}

func stringToTime(s, f string) (time.Time, error) {
	format := strings.ToLower(f)
	if format == "" {
		return guessTime(s)
	}
	if layout, ok := layouts[format]; ok {
		return time.Parse(layout, s)
	}

	switch format {
	case "unix":
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", s)
		}
		return time.UnixMicro(int64(f * 1000000)), nil
	case "unix-milli":
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", s)
		}
		return time.UnixMicro(int64(f * 1000)), nil
	case "unix-micro":
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", s)
		}
		return time.UnixMicro(int64(f)), nil
	default:
		return time.Time{}, fmt.Errorf("failed to parse time: %s", s)
	}
}

func timeToString(t time.Time, f string) string {
	format := strings.ToLower(f)
	if layout, ok := layouts[format]; ok {
		return t.Format(layout)
	}

	switch format {
	case "unix":
		f := float64(t.UnixNano()) / 1000000000
		return strconv.FormatFloat(f, 'f', -1, 64)
	case "unix-milli":
		f := float64(t.UnixNano()) / 1000000
		return strconv.FormatFloat(f, 'f', -1, 64)
	case "unix-micro":
		f := float64(t.UnixNano()) / 1000
		return strconv.FormatFloat(f, 'f', -1, 64)
	default:
		return format
	}
}

func guessTime(s string) (time.Time, error) {
	for _, rule := range guessRules {
		if rule.re.MatchString(s) {
			for _, l := range rule.layouts {
				if t, err := stringToTime(s, l); err == nil {
					return t, nil
				}
			}
		}
	}
	return time.Time{}, fmt.Errorf("Unknown format: %s", s)
}

func modifyTime(t time.Time, loc *time.Location, add, sub time.Duration) time.Time {
	t = t.In(loc)
	t = t.Add(add)
	t = t.Add(sub * -1)
	return t
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
		fmt.Fprint(&message, "")
		fmt.Fprint(&message, helpText)
		fmt.Fprintln(os.Stdout, message.String())
		os.Exit(0)
	}

	add := parseDuration(opts.Add)
	sub := parseDuration(opts.Sub)

	loc := time.Local
	if opts.Loc != "" {
		if v, err := time.LoadLocation(opts.Loc); err != nil {
			return err
		} else {
			loc = v
		}
	}

	var pat *regexp.Regexp
	if opts.Pattern != "" {
		if v, err := regexp.Compile(opts.Pattern); err != nil {
			return err
		} else {
			pat = v
		}
	}

	if opts.Now {
		t := time.Now()
		t = modifyTime(t, loc, add, sub)
		fmt.Println(timeToString(t, opts.Out))
	} else {
		scanner := genScanner(args)
		for scanner.Scan() {
			line := scanner.Text()
			if opts.Pattern == "" {
				t, err := stringToTime(strings.TrimSpace(line), opts.In)
				if err != nil {
					return err
				}

				t = modifyTime(t, loc, add, sub)
				fmt.Println(timeToString(t, opts.Out))
			} else {
				replaced := pat.ReplaceAllStringFunc(line, func(s string) string {
					t, err := stringToTime(s, opts.In)
					if err != nil {
						return s
					}

					t = modifyTime(t, loc, add, sub)
					return timeToString(t, opts.Out)
				})
				fmt.Println(replaced)
			}
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
