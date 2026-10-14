package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Run the CLI in a subprocess so flag parsing, stdin, and exit codes are tested
// without changing the parent process's global flags or standard streams.
func TestTimeconvCLIProcess(t *testing.T) {
	if os.Getenv("TIMECONV_CLI_TEST") != "1" {
		return
	}
	os.Args = append([]string{"timeconv"}, os.Args[3:]...)
	main()
	os.Exit(0)
}

func TestCLINow(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		stdin   string
		want    []string
		wantErr string
		unix    bool
		help    bool
	}{
		{name: "argument", args: []string{"now"}, want: []string{"now"}},
		{name: "numeric output location", args: []string{"now", "--location", "+09:00"}, want: []string{"now"}},
		{name: "case whitespace and mixed inputs", args: []string{"NOW", "1698292629", " \tNow \t", "now"}, want: []string{"now", "2023-10-26T13:56:09+09:00", "now", "now"}},
		{name: "stdin", stdin: "now\nNOW\n \tNow \t\n", want: []string{"now", "now", "now"}},
		{name: "explicit input format", args: []string{"--in", "unix", "now", "1698292629"}, want: []string{"now", "2023-10-26T13:56:09+09:00"}},
		{name: "unix output", args: []string{"--out", "unix", "now"}, want: []string{"now"}, unix: true},
		{name: "whole input only", args: []string{"--strict", "I have a plan now"}, wantErr: "line 1:"},
		{name: "grep", args: []string{"--grep", "(?i)now", "--in", "rfc3339", "--quiet", "I have a plan NOW"}, want: []string{"I have a plan NOW"}},
		{name: "replace", args: []string{"--replace", "rfc3339", "NOW 2023-10-26T03:57:09Z"}, want: []string{"NOW 2023-10-26T13:56:09+09:00"}},
		{name: "removed long flag", args: []string{"--now"}, wantErr: "unknown flag: --now"},
		{name: "removed short flag", args: []string{"-n"}, wantErr: "unknown shorthand flag: 'n'"},
		{name: "short help", args: []string{"-h"}, help: true},
		{name: "long help", args: []string{"--help"}, help: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"-test.run=^TestTimeconvCLIProcess$", "--", "--location", "Asia/Tokyo", "--out", "rfc3339nano", "--add", "1h", "--sub", "1m"}
			cmd := exec.Command(os.Args[0], append(args, tt.args...)...)
			cmd.Env = append(os.Environ(), "TIMECONV_CLI_TEST=1")
			cmd.Stdin = strings.NewReader(tt.stdin)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			before := time.Now().Add(59 * time.Minute)
			err := cmd.Run()
			after := time.Now().Add(59 * time.Minute)
			if tt.help {
				if err != nil || stdout.Len() != 0 || strings.Contains(stderr.String(), "--now") ||
					!strings.Contains(stderr.String(), "now --location +05:30") || !strings.Contains(stderr.String(), "--input-location +09:00 --location UTC") {
					t.Fatalf("error = %v, stdout = %q, stderr = %q; want help describing now and numeric timezone offsets", err, stdout.String(), stderr.String())
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(stderr.String(), tt.wantErr) || stdout.Len() != 0 {
					t.Fatalf("error = %v, stdout = %q, stderr = %q; want failure containing %q", err, stdout.String(), stderr.String(), tt.wantErr)
				}
				return
			}
			if err != nil || stderr.Len() != 0 {
				t.Fatalf("error = %v, stderr = %q", err, stderr.String())
			}
			lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
			if len(lines) != len(tt.want) {
				t.Fatalf("stdout = %q, want %d lines", stdout.String(), len(tt.want))
			}
			var firstNow string
			for i, want := range tt.want {
				if want != "now" {
					if lines[i] != want {
						t.Fatalf("line %d = %q, want %q", i+1, lines[i], want)
					}
					continue
				}
				if tt.unix {
					seconds, err := strconv.ParseFloat(lines[i], 64)
					if err != nil || seconds < float64(before.UnixMicro())/1e6 || seconds > float64(after.UnixMicro())/1e6 {
						t.Fatalf("Unix output %q outside current-time bounds: %v", lines[i], err)
					}
				} else {
					parsed, err := time.Parse(time.RFC3339Nano, lines[i])
					if err != nil || parsed.Before(before) || parsed.After(after) {
						t.Fatalf("output %q outside current-time bounds: %v", lines[i], err)
					}
					if _, offset := parsed.Zone(); offset != 9*60*60 {
						t.Fatalf("output offset = %d, want Asia/Tokyo", offset)
					}
				}
				if firstNow != "" && lines[i] != firstNow {
					t.Fatalf("now outputs differ: %q and %q", firstNow, lines[i])
				}
				firstNow = lines[i]
			}
		})
	}
}

func TestLocationValueOffsets(t *testing.T) {
	tests := []struct {
		value  string
		offset int
	}{
		{"+09:00", 9 * 3600},
		{"-09:00", -9 * 3600},
		{"+05:30", 5*3600 + 30*60},
		{"-03:30", -(3*3600 + 30*60)},
		{"+05:45", 5*3600 + 45*60},
		{"-00:45", -45 * 60},
		{"+00:00", 0},
		{"-00:00", 0},
		{"+00:01", 60},
		{"-00:01", -60},
		{"+23:00", 23 * 3600},
		{"+00:59", 59 * 60},
		{"+23:59", 23*3600 + 59*60},
		{"-23:59", -(23*3600 + 59*60)},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			var loc locationValue
			if err := loc.Set(tt.value); err != nil {
				t.Fatal(err)
			}
			if loc.String() != tt.value {
				t.Fatalf("String() = %q, want %q", loc.String(), tt.value)
			}
			for _, date := range []time.Time{
				time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
			} {
				name, offset := date.In(loc.Location).Zone()
				if name != tt.value || offset != tt.offset {
					t.Fatalf("Zone() at %s = (%q, %d), want (%q, %d)", date, name, offset, tt.value, tt.offset)
				}
			}
		})
	}
}

func TestLocationValueInvalidOffsets(t *testing.T) {
	for _, value := range []string{
		"+", "-", "+9", "+0900", "-0900", "+9:00", "+09:0", "+009:00", "+09:000",
		"+09:00:00", "+09:00 ", "+09:00\n", "+09.00", "+ab:cd", "+0a:00", "+09:0a",
		"++9:00", "--9:00", "+09:+0", "+０9:00", "+24:00", "-24:00", "+09:60", "-00:60", "+99:99",
	} {
		t.Run(value, func(t *testing.T) {
			loc := locationValue{Location: time.UTC}
			err := loc.Set(value)
			if err == nil || !strings.Contains(err.Error(), "expected ±HH:MM") {
				t.Fatalf("Set(%q) error = %v, want expected-format error", value, err)
			}
			if loc.Location != time.UTC {
				t.Fatal("invalid value changed the location")
			}
		})
	}
}

func TestLocationValueNamedLocations(t *testing.T) {
	for _, value := range []string{"", "UTC", "Local", "Asia/Tokyo", "America/New_York", "UTC+09:00", "09:00", "Invalid/Location"} {
		t.Run(value, func(t *testing.T) {
			want, wantErr := time.LoadLocation(value)
			loc := locationValue{Location: time.UTC}
			err := loc.Set(value)
			if wantErr != nil {
				if err == nil || err.Error() != wantErr.Error() || loc.Location != time.UTC {
					t.Fatalf("Set(%q) error = %v, want %v with unchanged location", value, err, wantErr)
				}
				return
			}
			if err != nil || loc.String() != want.String() {
				t.Fatalf("Set(%q) = %v, %v; want %v", value, loc.Location, err, want)
			}
			for _, month := range []time.Month{time.January, time.July} {
				date := time.Date(2026, month, 1, 0, 0, 0, 0, time.UTC)
				name, offset := date.In(loc.Location).Zone()
				wantName, wantOffset := date.In(want).Zone()
				if name != wantName || offset != wantOffset {
					t.Fatalf("Zone() = (%q, %d), want (%q, %d)", name, offset, wantName, wantOffset)
				}
			}
		})
	}
}

func TestCLILocationOffsets(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		want       string
		wantErr    string
		wantStderr string
	}{
		{name: "output", args: []string{"--location", "+05:30", "2026-10-13T00:00:00Z"}, want: "2026-10-13T05:30:00+05:30\n"},
		{name: "negative shorthand", args: []string{"-l", "-03:30", "2026-10-13T00:00:00Z"}, want: "2026-10-12T20:30:00-03:30\n"},
		{name: "deprecated alias", args: []string{"--loc", "+05:45", "2026-10-13T00:00:00Z"}, want: "2026-10-13T05:45:00+05:45\n", wantStderr: "Flag --loc has been deprecated"},
		{name: "input auto", args: []string{"--input-location", "+09:00", "--location", "UTC", "2026-10-13 09:00:00"}, want: "2026-10-13T00:00:00Z\n"},
		{name: "input named format", args: []string{"--in", "datetime", "--input-location", "-03:30", "--location", "UTC", "2026-10-13 09:00:00"}, want: "2026-10-13T12:30:00Z\n"},
		{name: "explicit offset wins", args: []string{"--input-location", "+09:00", "--location", "UTC", "2026-10-13T09:00:00+05:45"}, want: "2026-10-13T03:15:00Z\n"},
		{name: "explicit UTC wins", args: []string{"--in", "rfc3339", "--input-location", "+09:00", "--location", "UTC", "2026-10-13T09:00:00Z"}, want: "2026-10-13T09:00:00Z\n"},
		{name: "both offsets", args: []string{"--input-location", "+05:45", "--location", "-03:30", "2026-10-13 09:00:00"}, want: "2026-10-12T23:45:00-03:30\n"},
		{name: "zero zone name", args: []string{"--location", "+00:00", "--out", "MST", "2026-10-13T00:00:00Z"}, want: "+00:00\n"},
		{name: "negative zero zone name", args: []string{"--location", "-00:00", "--out", "MST", "2026-10-13T00:00:00Z"}, want: "-00:00\n"},
		{name: "invalid output", args: []string{"--location", "+24:00", "now"}, wantErr: "expected ±HH:MM"},
		{name: "invalid input", args: []string{"--input-location", "+09:60", "now"}, wantErr: "expected ±HH:MM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"-test.run=^TestTimeconvCLIProcess$", "--"}, tt.args...)
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "TIMECONV_CLI_TEST=1")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if tt.wantErr != "" {
				if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 || !strings.Contains(stderr.String(), tt.wantErr) || stdout.Len() != 0 {
					t.Fatalf("error = %v, stdout = %q, stderr = %q; want flag error containing %q", err, stdout.String(), stderr.String(), tt.wantErr)
				}
				return
			}
			if err != nil || stdout.String() != tt.want || (tt.wantStderr == "" && stderr.Len() != 0) || !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Fatalf("error = %v, stdout = %q, stderr = %q; want %q and diagnostic %q", err, stdout.String(), stderr.String(), tt.want, tt.wantStderr)
			}
		})
	}
}

func TestProcessTimeString(t *testing.T) {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	opts := options{
		in: "unix", out: "rfc3339",
		loc: locationValue{Location: jst}, inLoc: locationValue{Location: time.UTC},
		add: time.Hour, sub: time.Minute,
	}
	want := "2023-10-26T13:56:09+09:00"
	got, err := processTimeString("1698292629", &opts)
	if err != nil {
		t.Fatalf("processTimeString failed: %v", err)
	}
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGenScannerAcceptsLongLines(t *testing.T) {
	line := strings.Repeat("x", 128*1024)
	scanner := genScanner([]string{line})
	if !scanner.Scan() {
		t.Fatalf("Scan failed: %v", scanner.Err())
	}
	if scanner.Text() != line {
		t.Fatalf("scanned line length = %d, want %d", len(scanner.Text()), len(line))
	}
	if scanner.Scan() || scanner.Err() != nil {
		t.Fatalf("unexpected second scan result: %v", scanner.Err())
	}
}

func TestScannerLimit(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader(strings.Repeat("x", maxScanTokenSize+1)))
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)
	if scanner.Scan() || scanner.Err() == nil {
		t.Fatal("expected an over-limit scanner error")
	}
}

func TestValidateOptionsReplace(t *testing.T) {
	tests := []struct {
		name    string
		opts    options
		wantErr bool
		wantIn  string
	}{
		{name: "implicit input", opts: options{replace: "UNIX-MILLI", replaceSet: true}, wantIn: "unix-milli"},
		{name: "matching input", opts: options{replace: "unix", replaceSet: true, in: "UNIX", inputSet: true}, wantIn: "unix"},
		{name: "conflicting input", opts: options{replace: "unix-milli", replaceSet: true, in: "unix", inputSet: true}, wantErr: true},
		{name: "grep conflict", opts: options{replace: "unix", replaceSet: true, grepSet: true}, wantErr: true},
		{name: "unknown format", opts: options{replace: "epoch", replaceSet: true}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.opts
			err := validateOptions(&opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && opts.in != tt.wantIn {
				t.Fatalf("input format = %q, want %q", opts.in, tt.wantIn)
			}
		})
	}
}

func TestReplaceNamedTimestamps(t *testing.T) {
	tests := []struct {
		format string
		input  string
	}{
		{"ansic", "Mon Jan  2 15:04:05 2006"},
		{"unixdate", "Mon Jan  2 15:04:05 UTC 2006"},
		{"rubydate", "Mon Jan 02 15:04:05 +0000 2006"},
		{"rfc822", "02 Jan 06 15:04 UTC"},
		{"rfc822z", "02 Jan 06 15:04 +0000"},
		{"rfc850", "Monday, 02-Jan-06 15:04:05 UTC"},
		{"rfc1123", "Mon, 02 Jan 2006 15:04:05 UTC"},
		{"rfc1123z", "Mon, 02 Jan 2006 15:04:05 +0000"},
		{"rfc3339", "2006-01-02T15:04:05Z"},
		{"rfc3339nano", "2006-01-02T15:04:05.123456789Z"},
		{"kitchen", "3:04PM"},
		{"stamp", "Jan  2 15:04:05"},
		{"stampmilli", "Jan  2 15:04:05.123"},
		{"stampmicro", "Jan  2 15:04:05.123456"},
		{"stampnano", "Jan  2 15:04:05.123456789"},
		{"datetime", "2006-01-02 15:04:05"},
		{"dateonly", "2006-01-02"},
		{"timeonly", "15:04:05"},
		{"unix", "1136239445.25"},
		{"unix-milli", "1136239445250"},
		{"unix-micro", "1136239445250000"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			opts := &options{
				in: tt.format, out: "2006", replace: tt.format,
				loc: locationValue{Location: time.UTC}, inLoc: locationValue{Location: time.UTC},
			}
			line := "before [" + tt.input + "] after"
			got, err := replaceNamedTimestamps(line, opts)
			if err != nil {
				t.Fatalf("replaceNamedTimestamps returned error: %v", err)
			}
			if got == line || !strings.HasPrefix(got, "before [") || !strings.HasSuffix(got, "] after") {
				t.Fatalf("unexpected replacement: %q", got)
			}
		})
	}
}

func TestReplaceNamedTimestampsMultipleValues(t *testing.T) {
	opts := &options{
		in: "unix-milli", out: "rfc3339nano", replace: "unix-milli",
		loc: locationValue{Location: time.UTC}, inLoc: locationValue{Location: time.UTC},
	}
	got, err := replaceNamedTimestamps("a=1698292629955 b=1698292630057", opts)
	if err != nil {
		t.Fatalf("replaceNamedTimestamps returned error: %v", err)
	}
	want := "a=2023-10-26T03:57:09.955Z b=2023-10-26T03:57:10.057Z"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReplaceNamedTimestampsDoesNotPartiallyMatchLongerEpoch(t *testing.T) {
	opts := &options{
		in: "unix-milli", out: "rfc3339", replace: "unix-milli",
		loc: locationValue{Location: time.UTC}, inLoc: locationValue{Location: time.UTC},
	}
	line := "id=1698292629955123"
	got, err := replaceNamedTimestamps(line, opts)
	if err != nil {
		t.Fatalf("replaceNamedTimestamps returned error: %v", err)
	}
	if got != line {
		t.Fatalf("got %q, want unchanged input", got)
	}
}

func TestReplaceNamedTimestampsReturnsParseErrorAndPreservesValue(t *testing.T) {
	opts := &options{
		in: "dateonly", out: "unix", replace: "dateonly",
		loc: locationValue{Location: time.UTC}, inLoc: locationValue{Location: time.UTC},
	}
	line := "date=2023-19-32"
	got, err := replaceNamedTimestamps(line, opts)
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if got != line {
		t.Fatalf("got %q, want unchanged input", got)
	}
}

func TestReplaceNamedTimestampsDoesNotMatchLowerPrecisionPrefix(t *testing.T) {
	opts := &options{
		in: "stamp", out: "unix", replace: "stamp",
		loc: locationValue{Location: time.UTC}, inLoc: locationValue{Location: time.UTC},
	}
	line := "at=Jan  2 15:04:05.123"
	got, err := replaceNamedTimestamps(line, opts)
	if err != nil {
		t.Fatalf("replaceNamedTimestamps returned error: %v", err)
	}
	if got != line {
		t.Fatalf("got %q, want unchanged input", got)
	}
}
