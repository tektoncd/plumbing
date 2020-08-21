package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

const expectedDateFormat = "2006-01-02"
const usageMessage = `
generate-rotation-csv prints a csv-formatted rotation from a
comma-separated list of names. Weekend days are considered to
be Saturday and Sunday and are skipped, meaning a blank string
is inserted in place of a user's name.

The printed csv has two columns: Date and User.
`

// config is the set of options to configure printing of the rotation.
type config struct {
	// the complete list of names to print in the order they should appear
	names []string
	// the name to start from in the list of names
	startName string
	// the date to start printing from
	startDate time.Time
	// the number of days to print entries for
	days uint
	// a map of dates to usernames that override whatever would be generated
	overrides map[string]string
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	c, err := parseFlags()
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	err = validateConfig(c)
	if err != nil {
		log.Fatalf("error validating config: %v", err)
	}

	err = generateRotationCSV(c, os.Stdout)
	if err != nil {
		log.Fatalf("error printing csv: %v", err)
	}
}

// generateRotationCSV writes a rotation starting at c.startName
// and c.startDate and then repeatedly iterating over c.names
// until it reaches c.startDate + c.days. Saturday and Sunday are
// left blank. The output is written to the provided io.Writer.
func generateRotationCSV(c config, out io.Writer) error {
	iter := c.startDate
	end := c.startDate.AddDate(0, 0, int(c.days))
	n := 0

	for i, name := range c.names {
		if name == c.startName {
			n = i
			break
		}
	}

	w := csv.NewWriter(out)
	w.Write([]string{"Date", "User"})
	for iter.Before(end) {
		if iter.Weekday() == time.Saturday || iter.Weekday() == time.Sunday {
			w.Write([]string{iter.Format(expectedDateFormat), ""})
			iter = iter.AddDate(0, 0, 1)
			continue
		}
		date := iter.Format(expectedDateFormat)
		user := c.names[n]
		if userOverride, ok := c.overrides[date]; ok {
			user = userOverride
		}
		if err := w.Write([]string{date, user}); err != nil {
			return err
		}
		n++
		if n >= len(c.names) {
			n = 0
		}
		iter = iter.AddDate(0, 0, 1)
	}
	w.Flush()
	return w.Error()
}

// validateConfig checks that c is suitably populated for generating a valid
// rotation.
func validateConfig(c config) error {
	names := make(map[string]struct{})
	for _, n := range c.names {
		names[n] = struct{}{}
	}
	if c.startName == "" {
		return errors.New("missing starting name")
	}
	if _, ok := names[c.startName]; !ok {
		return errors.New("starting name does not appear in list of user names")
	}
	if len(c.overrides) != 0 {
		for date, name := range c.overrides {
			if name == "" { // blank name indicates nobody should be on rotation that day
				continue
			}
			if _, ok := names[name]; !ok {
				return fmt.Errorf("override %s,%s: user does not appear in list of user names", date, name)
			}
		}
	}
	return nil
}

// parseFlags generates a config from CLI flags that the user has passed in,
// populating certain missing portions of config with sane defaults.
func parseFlags() (config, error) {
	var c config
	namesInput := ""
	startDateInput := ""
	defaultStartDate := time.Now().Format(expectedDateFormat)
	overridesPath := ""

	flag.StringVar(&namesInput, "names", "", "comma-separated list of user names to generate rotation from.")
	flag.StringVar(&c.startName, "start-name", "", "the user name to begin the rotation. (defaults to the first provided name in -names list)")
	flag.StringVar(&startDateInput, "start-date", defaultStartDate, "date to begin the rotation from in YYYY-MM-DD format. starts today if left blank.")
	flag.UintVar(&c.days, "days", 365, "number of days to run the rotation for.")
	flag.StringVar(&overridesPath, "overrides", "", "path to a csv file in YYYY-MM-DD,user format that should override whatever this program generates for that date.")
	flag.Usage = func() {
		log.Printf(strings.TrimSpace(usageMessage) + "\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ns := strings.Split(namesInput, ",")
	for _, n := range ns {
		t := strings.TrimSpace(n)
		if t != "" {
			c.names = append(c.names, t)
		}
	}

	d, err := time.Parse(expectedDateFormat, startDateInput)
	if err != nil {
		return c, fmt.Errorf("error parsing start date of %q: %v", startDateInput, err)
	}
	c.startDate = d

	if c.startName == "" && len(c.names) > 0 {
		c.startName = c.names[0]
	}

	if overridesPath != "" {
		f, err := os.Open(overridesPath)
		if err != nil {
			return c, fmt.Errorf("unable to open overrides %q: %v", overridesPath, err)
		}
		c.overrides, err = loadOverrides(f)
		if err != nil {
			return c, err
		}
	}

	return c, nil
}

// loadOverrides parses records from the in io.Reader and returns the parsed contents
// as a map from string date to string user name. Each record in the csv is expected to be
// 2 or more fields. Field 1 is expected to be a date, Field 2 is expected to be a
// username and Fields 3+ can be comments or anything else.
func loadOverrides(in io.Reader) (map[string]string, error) {
	r := csv.NewReader(in)
	out := make(map[string]string)
	n := 0
	for {
		n++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if errors.Is(err, csv.ErrFieldCount) && len(record) < 2 {
				return out, err
			}
			if !errors.Is(err, csv.ErrFieldCount) {
				return out, err
			}
		}
		if n == 1 && strings.ToLower(record[0]) == "date" {
			continue
		}
		out[record[0]] = record[1]
	}
	return out, nil
}
