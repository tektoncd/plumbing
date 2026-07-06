# generate-rotation-csv

generate-rotation-csv prints a csv-formatted rotation from a
comma-separated list of names. Weekend days are considered to
be Saturday and Sunday and are skipped, meaning a blank string
is inserted in place of a user's name.

## Usage

Generate a rotation CSV starting today for one year:
```bash
go run ./main.go -names jules,ronda,gavin,paula,don,carol
```

Generate a rotation CSV starting tomorrow (today is 2020-02-28) for 6 weeks:
```bash
go run ./main.go -start-date 2020-02-29 -days $((6 * 7)) -names jules,ronda,gavin,paula,don,carol
```

Generate a rotation CSV starting with carol yesterday (today is 2020-02-28) for 165 weeks:
```bash
go run ./main.go -start-date 2020-02-27 -days $((165 * 7)) -start-name carol -names jules,ronda,gavin,paula,don,carol
```

Print help
```bash
go run ./main.go -h
```

## Flags

  -names string
    	comma-separated list of user names to generate rotation from. required.
  -days uint
    	number of days to run the rotation for. defaults to 1 year.
  -start-date string
    	date to begin the rotation from in YYYY-MM-DD format. starts today if left blank.
  -start-name string
    	the user name to begin the rotation. defaults to the first provided name in -names list
  -overrides string
    	path to a csv file in YYYY-MM-DD,user format that should override whatever this program generates for that date.

## Overrides

Specify an overrides file with the -overrides flag. This allows
for rotation swaps and global holidays to be recorded somewhere.

Each override entry will replace whatever value is generated
for a given date.

Th file should be CSV with each record having at least two fields, one
for date and one for user name.

Override entries can include extra fields for comments or other
additional info. Extra fields will be ignored.

Here's an example overrides.csv for the holiday season of 2020. bob
swaps with sbws on 23rd / 24th and then nobody is on rotation from
24th Dec until 2nd of Jan, when the rotation will resume:

```
Date,User
2020-12-23,bob,Swapping with sbws tomorrow
2020-12-24,sbws,Covering for bobs first day of vacation
2020-12-25,
2020-12-26,
2020-12-27,
2020-12-28,
2020-12-29,
2020-12-30,
2020-12-31,
2020-1-1,,Happy New Year!
```

## Example Output

```bash
$ go run ./main.go -names bob,carol,donna,jules -days 8 -start-date 1991-01-01
Date,User
1991-01-01,bob
1991-01-02,carol
1991-01-03,donna
1991-01-04,jules
1991-01-05,
1991-01-06,
1991-01-07,bob
1991-01-08,carol
```
