package main

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

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
