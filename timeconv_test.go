package timeconv

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name, input, format string
		want                time.Time
		wantErr             bool
	}{
		{"Unix", "1698292629", "unix", time.Unix(1698292629, 0), false},
		{"RFC3339", "2023-10-26T12:57:09+09:00", "rfc3339", time.Date(2023, 10, 26, 12, 57, 9, 0, time.FixedZone("", 9*60*60)), false},
		{"Auto-detect Unix", "1698292629", "", time.Unix(1698292629, 0), false},
		{"Auto-detect RFC3339", "2023-10-26T12:57:09Z", "", time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC), false},
		{"Invalid", "invalid time", "", time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Fatalf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name, format, want string
		input              time.Time
	}{
		{"Unix", "unix", "1698292629", time.Unix(1698292629, 0)},
		{"RFC3339", "rfc3339", "2023-10-26T12:57:09Z", time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC)},
		{"Custom", "2006-01-02", "2023-10-26", time.Date(2023, 10, 26, 12, 57, 9, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Format(tt.input, tt.format); got != tt.want {
				t.Fatalf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAutoDetectsEpochUnits(t *testing.T) {
	want := time.Unix(1698292629, 955123000)
	for _, input := range []string{"1698292629.955123", "1698292629955.123", "1698292629955123"} {
		t.Run(input, func(t *testing.T) {
			got, err := Parse(input, "")
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if !got.Equal(want) {
				t.Fatalf("Parse(%q) = %v, want %v", input, got, want)
			}
		})
	}
}

func TestParseInLocation(t *testing.T) {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, input, format string
		want                time.Time
	}{
		{"zoneless input", "2026-09-30 12:00:00", "datetime", time.Date(2026, 9, 30, 12, 0, 0, 0, jst)},
		{"explicit zone", "2026-09-30T12:00:00Z", "rfc3339", time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInLocation(tt.input, tt.format, jst)
			if err != nil {
				t.Fatalf("ParseInLocation returned error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
