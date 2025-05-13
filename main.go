package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

const layoutExamples = `  ANSIC       "Mon Jan _2 15:04:05 2006"
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

var epochLayouts = map[string]int64{
	"unix":       1e6,
	"unix-milli": 1e3,
	"unix-micro": 1,
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

type options struct {
	in     string
	out    string
	now    bool
	add    time.Duration
	sub    time.Duration
	loc    locationValue
	re     regexpValue
	inputs []string
}

func parseFlags() *options {
	var opts options
	opts.loc.Location = time.Local

	pflag.StringVarP(&opts.in, "in", "i", "", "Input time format (default: auto)")
	pflag.StringVarP(&opts.out, "out", "o", "rfc3339", "Output time format")
	pflag.BoolVarP(&opts.now, "now", "n", false, "Load current time as input")
	pflag.DurationVarP(&opts.add, "add", "a", time.Duration(0), "Append time duration (ex. 5m, 1.5h, 1h30m)")
	pflag.DurationVarP(&opts.sub, "sub", "s", time.Duration(0), "Substruct time duration (ex. 5m, 1.5h, 1h30m)")
	pflag.VarP(&opts.loc, "location", "l", "Timezone location (e.g., UTC, Asia/Tokyo)")
	pflag.VarP(&opts.re, "grep", "g", "Replace strings that match the regular expression")
	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintf(os.Stderr, "  %s [Options] [file...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintf(os.Stderr, "%s\n", pflag.CommandLine.FlagUsages())
		fmt.Fprintln(os.Stderr, "Format Examples:")
		fmt.Fprintf(os.Stderr, "%s\n", layoutExamples)
		os.Exit(0)
	}

	pflag.Parse()

	opts.inputs = pflag.Args()
	opts.in = strings.ToLower(opts.in)
	opts.out = strings.ToLower(opts.out)
	return &opts
}

func stringToTime(s, format string) (time.Time, error) {
	if format == "" {
		return guessTime(s)
	}

	if scale, ok := epochLayouts[format]; ok {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse epoch time: %s", s)
		}
		return time.UnixMicro(int64(v * float64(scale))), nil
	}

	if layout, ok := knownLayouts[format]; ok {
		return time.Parse(layout, s)
	}
	return time.Time{}, fmt.Errorf("failed to parse time: %s", s)
}

func timeToString(t time.Time, format string) string {
	if scale, ok := epochLayouts[format]; ok {
		v := float64(t.UnixMicro())
		return strconv.FormatFloat(v/float64(scale), 'f', -1, 64)
	}

	if layout, ok := knownLayouts[format]; ok {
		return t.Format(layout)
	}
	return t.Format(format)
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

func genScanner(args []string) *bufio.Scanner {
	if len(args) > 0 {
		reader := strings.NewReader(strings.Join(args, "\n"))
		return bufio.NewScanner(reader)
	}
	return bufio.NewScanner(os.Stdin)
}

func modifyTime(t time.Time, loc locationValue, add, sub time.Duration) time.Time {
	t = t.In(loc.Location)
	t = t.Add(add)
	t = t.Add(sub * -1)
	return t
}

type locationValue struct {
	*time.Location
}

func (lv *locationValue) String() string {
	return lv.Location.String()
}

func (lv *locationValue) Set(value string) error {
	loc, err := time.LoadLocation(value)
	if err != nil {
		return fmt.Errorf("invalid location %q: %w", value, err)
	}
	lv.Location = loc
	return nil
}

func (lv *locationValue) Type() string {
	return "location"
}

type regexpValue struct {
	*regexp.Regexp
}

func (rv *regexpValue) String() string {
	if rv.Regexp == nil {
		return ""
	}
	return rv.Regexp.String()
}

func (rv *regexpValue) Set(s string) error {
	if s == "" {
		re, err := regexp.Compile(s)
		if err != nil {
			return err
		}
		rv.Regexp = re
	}
	return nil
}

func (rv *regexpValue) Type() string {
	return "regexp"
}

func run() error {
	opts := parseFlags()

	if opts.now {
		t := time.Now()
		t = modifyTime(t, opts.loc, opts.add, opts.sub)
		fmt.Println(timeToString(t, opts.out))
	} else {
		scanner := genScanner(opts.inputs)
		for scanner.Scan() {
			line := scanner.Text()
			if opts.re.Regexp == nil {
				t, err := stringToTime(strings.TrimSpace(line), opts.in)
				if err != nil {
					return err
				}

				t = modifyTime(t, opts.loc, opts.add, opts.sub)
				fmt.Println(timeToString(t, opts.out))
			} else {
				replaced := opts.re.ReplaceAllStringFunc(line, func(s string) string {
					t, err := stringToTime(s, opts.in)
					if err != nil {
						return s
					}

					t = modifyTime(t, opts.loc, opts.add, opts.sub)
					return timeToString(t, opts.out)
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
