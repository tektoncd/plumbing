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
