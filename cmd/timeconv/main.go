package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/haccht/timeconv"
	"github.com/spf13/pflag"
)

const maxScanTokenSize = 1024 * 1024

type options struct {
	in, out       string
	replace       string
	now           bool
	add, sub      time.Duration
	loc, inLoc    locationValue
	re            regexpValue
	strict, quiet bool
	inputs        []string
	inputSet      bool
	grepSet       bool
	replaceSet    bool
}

func parseFlags() *options {
	var opts options
	opts.loc.Location = time.Local
	opts.inLoc.Location = time.UTC
	pflag.StringVarP(&opts.in, "in", "i", "", "Input time format (default: auto)")
	pflag.StringVarP(&opts.out, "out", "o", "rfc3339", "Output time format")
	pflag.BoolVarP(&opts.now, "now", "n", false, "Load current time as input")
	pflag.DurationVarP(&opts.add, "add", "a", 0, "Append time duration (ex. 5m, 1.5h, 1h30m)")
	pflag.DurationVarP(&opts.sub, "sub", "s", 0, "Substruct time duration (ex. 5m, 1.5h, 1h30m)")
	pflag.VarP(&opts.loc, "location", "l", "Output timezone location (e.g., UTC, Asia/Tokyo)")
	pflag.Var(&opts.loc, "loc", "Alias for --location")
	_ = pflag.CommandLine.MarkDeprecated("loc", "use --location instead")
	_ = pflag.CommandLine.MarkHidden("loc")
	pflag.Var(&opts.inLoc, "input-location", "Timezone for inputs without an explicit timezone")
	pflag.VarP(&opts.re, "grep", "g", "Replace strings that match the regular expression")
	pflag.StringVarP(&opts.replace, "replace", "G", "", "Replace timestamps matching a named input format")
	pflag.BoolVar(&opts.strict, "strict", false, "Stop at the first invalid timestamp")
	pflag.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress invalid timestamp diagnostics")
	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintf(os.Stderr, "  %s [Options] [time...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintf(os.Stderr, "%s\n", pflag.CommandLine.FlagUsages())
		fmt.Fprintln(os.Stderr, "Format Examples:")
		fmt.Fprintf(os.Stderr, "%s\n", timeconv.LayoutExamples)
		os.Exit(0)
	}
	pflag.Parse()
	opts.inputs = pflag.Args()
	opts.inputSet = pflag.CommandLine.Changed("in")
	opts.grepSet = pflag.CommandLine.Changed("grep")
	opts.replaceSet = pflag.CommandLine.Changed("replace")
	return &opts
}

func validateOptions(opts *options) error {
	if opts.grepSet && opts.replaceSet {
		return fmt.Errorf("--grep and --replace cannot be used together")
	}
	if !opts.replaceSet {
		return nil
	}
	format := strings.ToLower(opts.replace)
	if _, err := timeconv.FindAllInLocation("", format, opts.inLoc.Location); err != nil {
		return fmt.Errorf("unsupported replace format: %s", opts.replace)
	}
	if opts.inputSet && !strings.EqualFold(opts.in, format) {
		return fmt.Errorf("--in %s conflicts with --replace %s", opts.in, opts.replace)
	}
	opts.replace = format
	opts.in = format
	return nil
}

func genScanner(args []string) *bufio.Scanner {
	var scanner *bufio.Scanner
	if len(args) > 0 {
		scanner = bufio.NewScanner(strings.NewReader(strings.Join(args, "\n")))
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)
	return scanner
}

func modifyTime(t time.Time, loc locationValue, add, sub time.Duration) time.Time {
	if t.Year() == 0 {
		now := time.Now().In(t.Location())
		t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	}
	return t.In(loc.Location).Add(add).Add(-sub)
}

type locationValue struct{ *time.Location }

func (lv *locationValue) String() string {
	if lv.Location == nil {
		return ""
	}
	return lv.Location.String()
}
func (lv *locationValue) Set(value string) error {
	loc, err := time.LoadLocation(value)
	if err == nil {
		lv.Location = loc
	}
	return err
}
func (lv *locationValue) Type() string { return "location" }

type regexpValue struct{ *regexp.Regexp }

func (rv *regexpValue) String() string {
	if rv.Regexp == nil {
		return ""
	}
	return rv.Regexp.String()
}
func (rv *regexpValue) Set(value string) error {
	compiled, err := regexp.Compile(value)
	if err == nil {
		rv.Regexp = compiled
	}
	return err
}
func (rv *regexpValue) Type() string { return "regexp" }

func processTimeString(value string, opts *options) (string, error) {
	parsed, err := timeconv.ParseInLocation(value, opts.in, opts.inLoc.Location)
	if err != nil {
		return "", err
	}
	return timeconv.Format(modifyTime(parsed, opts.loc, opts.add, opts.sub), opts.out), nil
}

func reportError(err error, line string, lineNumber int, opts *options) error {
	wrapped := fmt.Errorf("line %d: %w: %q", lineNumber, err, line)
	if opts.strict {
		return wrapped
	}
	if !opts.quiet {
		fmt.Fprintln(os.Stderr, wrapped.Error())
	}
	return nil
}

func run() error {
	opts := parseFlags()
	if err := validateOptions(opts); err != nil {
		return err
	}
	if opts.now {
		fmt.Println(timeconv.Format(modifyTime(time.Now(), opts.loc, opts.add, opts.sub), opts.out))
		return nil
	}
	scanner := genScanner(opts.inputs)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if opts.replaceSet {
			replaced, replaceErr := replaceNamedTimestamps(line, opts)
			if replaceErr != nil {
				if err := reportError(replaceErr, line, lineNumber, opts); err != nil {
					return err
				}
			}
			fmt.Println(replaced)
			continue
		}
		if opts.re.Regexp == nil {
			out, err := processTimeString(strings.TrimSpace(line), opts)
			if err != nil {
				if err := reportError(err, line, lineNumber, opts); err != nil {
					return err
				}
				continue
			}
			fmt.Println(out)
			continue
		}
		var replaceErr error
		replaced := opts.re.ReplaceAllStringFunc(line, func(value string) string {
			out, err := processTimeString(value, opts)
			if err != nil {
				if replaceErr == nil {
					replaceErr = err
				}
				return value
			}
			return out
		})
		if replaceErr != nil {
			if err := reportError(replaceErr, line, lineNumber, opts); err != nil {
				return err
			}
		}
		fmt.Println(replaced)
	}
	return scanner.Err()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
