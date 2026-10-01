package timeconv

import (
	"testing"
	"time"
)

var benchmarkTime time.Time

func TestParse(t *testing.T) {
	tests := []struct {
		name, input, format string
		want                time.Time
		wantErr             bool
	}{
		{"Unix", "1698292629", "unix", time.Unix(1698292629, 0), false},
		{"RFC3339", "2023-10-26T12:57:09+09:00", "rfc3339", time.Date(2023, 10, 26, 12, 57, 9, 0, time.FixedZone("", 9*60*60)), false},
		{"Auto-detect Unix", "1698292629", "", time.Unix(1698292629, 0), false},
		{"Explicit auto", "1698292629", "auto", time.Unix(1698292629, 0), false},
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

func TestParseAutoDetectsNamedFormats(t *testing.T) {
	tests := []string{
		"Mon Jan  2 15:04:05 2006",
		"Mon Jan  2 15:04:05 UTC 2006",
		"Mon Jan 02 15:04:05 +0000 2006",
		"02 Jan 06 15:04 UTC",
		"02 Jan 06 15:04 +0000",
		"Monday, 02-Jan-06 15:04:05 UTC",
		"Mon, 02 Jan 2006 15:04:05 UTC",
		"Mon, 02 Jan 2006 15:04:05 +0000",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.123456789Z",
		"3:04PM",
		"Jan  2 15:04:05",
		"Jan  2 15:04:05.123",
		"Jan  2 15:04:05.123456",
		"Jan  2 15:04:05.123456789",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input, "auto"); err != nil {
				t.Fatalf("Parse(%q, auto) returned error: %v", input, err)
			}
		})
	}
}

func TestFindAllInLocation(t *testing.T) {
	input := "a=1698292629955 b=1698292630057"
	matches, err := FindAllInLocation(input, "UNIX-MILLI", time.UTC)
	if err != nil {
		t.Fatalf("FindAllInLocation returned error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("match count = %d, want 2", len(matches))
	}
	for i, want := range []string{"1698292629955", "1698292630057"} {
		if got := input[matches[i].Start:matches[i].End]; got != want {
			t.Fatalf("match %d = %q, want %q", i, got, want)
		}
	}
}

func TestFindAllInLocationRejectsPartialEpoch(t *testing.T) {
	matches, err := FindAllInLocation("id=1698292629955123", "unix-milli", time.UTC)
	if err != nil {
		t.Fatalf("FindAllInLocation returned error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("match count = %d, want 0", len(matches))
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

func BenchmarkParseAuto(b *testing.B) {
	tests := map[string]string{
		"Unix":        "1698292629.955123",
		"UnixMilli":   "1698292629955.123",
		"UnixMicro":   "1698292629955123",
		"RFC3339Nano": "2026-10-01T12:34:56.123456789+09:00",
		"DateTime":    "2026-10-01 12:34:56",
		"Stamp":       "Oct  1 12:34:56",
		"Invalid":     "not a timestamp",
	}
	for name, input := range tests {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkTime, _ = Parse(input, "auto")
			}
		})
	}
}
