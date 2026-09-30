# timeconv

`timeconv` converts time strings to a different format.

It can also be used as a Go library:

```go
parsed, err := timeconv.ParseInLocation(value, format, location)
formatted := timeconv.Format(parsed, "rfc3339")
```

```bash
$ timeconv -h
Usage:
  timeconv [Options] [time...]

Options:
  -i, --in string           Input time format (default: auto)
  -o, --out string          Output time format (default "rfc3339")
  -n, --now                 Load current time as input
  -a, --add duration        Append time duration (ex. 5m, 1.5h, 1h30m)
  -s, --sub duration        Substruct time duration (ex. 5m, 1.5h, 1h30m)
  -l, --location location   Output timezone location (e.g., UTC, Asia/Tokyo) (default Local)
      --input-location location
                            Timezone for inputs without an explicit timezone (default UTC)
  -g, --grep regexp         Replace strings that match the regular expression
      --strict              Stop at the first invalid timestamp
  -q, --quiet               Suppress invalid timestamp diagnostics

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

  Arbitrary formats are also supported. See https://pkg.go.dev/time as a reference.
```


`timeconv` accepts timestamp arguments as well as standard input. Positional
arguments are timestamp values, not file names.

```bash
$ timeconv --in unix 1698292629.955
2023-10-26T03:57:09Z

$ cat timeconvs.txt
1698292629.955
1698292630.057
1698288090.445

$ cat timeconvs.txt | timeconv --in unix
2023-10-26T03:57:09Z
2023-10-26T03:57:10Z
2023-10-26T02:41:30Z
```

`timeconv` can translate from a specific time format to another format.

```bash
$ echo 2023-11-01T09:00:00Z | timeconv --in rfc3339 --out '02 Jan 06 15:04 MST'
01 Nov 23 09:00 UTC
```

`timeconv` can also override the output timezone as well.

```bash
$ echo 2023-11-01T09:00:00Z | timeconv --in rfc3339 --out '02 Jan 06 15:04 MST' --location Asia/Tokyo
01 Nov 23 18:00 JST
```

For an input without a timezone, use `--input-location` to specify how it
should be interpreted. `--location` independently controls the output timezone.

```bash
$ timeconv --in datetime --input-location Asia/Tokyo --location UTC '2023-11-01 18:00:00'
2023-11-01T09:00:00Z
```

Malformed timestamps are reported with their line number and processing
continues by default. Use `--strict` to stop at the first error, or `--quiet` to
suppress these diagnostics.
