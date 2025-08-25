package main

import (
	"testing"
	"time"
)

func Test_processTimeString(t *testing.T) {
	jst, _ := time.LoadLocation("Asia/Tokyo")
	opts := options{
		in:  "unix",
		out: "rfc3339",
		loc: locationValue{Location: jst},
		add: time.Hour,
		sub: time.Minute,
	}

	// 1698292629 -> 2023-10-26 12:57:09 +0900 JST
	// add 1h -> 13:57:09
	// sub 1m -> 13:56:09
	// to rfc3339 -> 2023-10-26T13:56:09+09:00
	expected := "2023-10-26T13:56:09+09:00"
	actual, err := processTimeString("1698292629", &opts)
	if err != nil {
		t.Fatalf("processTimeString failed: %v", err)
	}

	if actual != expected {
		t.Errorf("expected %s, but got %s", expected, actual)
	}
}

func Test_stringToTime(t *testing.T) {
	type args struct {
		s      string
		format string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{"Unix", args{"1698292629", "unix"}, time.Unix(1698292629, 0), false},
		{"RFC3339", args{"2023-10-26T12:57:09+09:00", "rfc3339"}, time.Date(2023, 10, 26, 12, 57, 9, 0, time.FixedZone("", 9*60*60)), false},
		{"Auto-detect Unix", args{"1698292629", ""}, time.Unix(1698292629, 0), false},
		{"Auto-detect RFC3339", args{"2023-10-26T12:57:09Z", ""}, time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC), false},
		{"Invalid", args{"invalid time", ""}, time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stringToTime(tt.args.s, tt.args.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("stringToTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// We need to compare with a tolerance for timezone differences in auto-detection
			if !tt.wantErr && !got.Equal(tt.want) {
				// A simple Equal might fail if location is different (e.g. UTC vs Local)
				// So we check the unix timestamp for equality
				if got.Unix() != tt.want.Unix() {
					t.Errorf("stringToTime() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func Test_timeToString(t *testing.T) {
	type args struct {
		t      time.Time
		format string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Unix", args{time.Unix(1698292629, 0), "unix"}, "1698292629"},
		{"RFC3339", args{time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC), "rfc3339"}, "2023-10-26T12:57:09Z"},
		{"Custom", args{time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC), "2006-01-02"}, "2023-10-26"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeToString(tt.args.t, tt.args.format); got != tt.want {
				t.Errorf("timeToString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_guessTime(t *testing.T) {
	// This is implicitly tested by Test_stringToTime with auto-detection cases
	// but we can add a specific one for a tricky case.
	s := "Mon Jan 02 15:04:05 -0700 2006"
	expected, _ := time.Parse(time.RubyDate, s)
	actual, err := guessTime(s)
	if err != nil {
		t.Fatalf("guessTime failed: %v", err)
	}
	if !actual.Equal(expected) {
		t.Errorf("expected %v, but got %v", expected, actual)
	}
}
