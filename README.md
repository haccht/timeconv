# timeconv

`timeconv` converts any time string to a differenct format.

```bash
$ timeconv -h
Usage:
  timeconv [Options]

Application Options:
  -i, --in=   Specify input time format (default: Unix)
  -o, --out=  Specify output time format (default: RFC3339)
  -z, --tz=   Override timezone
  -h, --help  Show this help message

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


`timeconv` accepts argument variables as well as the standard input.

```bash
$ timeconv 1698292629.955
2023-10-26T03:57:09Z


$ cat timeconvs.txt
1698292629.955
1698292630.057
1698288090.445

$ cat timeconvs.txt | timeconv
2023-10-26T03:57:09Z
2023-10-26T03:57:10Z
2023-10-26T02:41:30Z
```

`timeconv` can translate from a specific time format to another format.

```bash
$ echo 2023-11-01T09:00:00Z | timeconv -i RFC3339 -o '02 Jan 06 15:04 MST'
01 Nov 23 09:00 UTC
```

`timeconv` can also override the output timezone as well.

```bash
$ echo 2023-11-01T09:00:00Z | timeconv -i RFC3339 -o '02 Jan 06 15:04 MST' -z Asia/Tokyo
01 Nov 23 18:00 JST
```
