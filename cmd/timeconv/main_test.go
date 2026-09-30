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
