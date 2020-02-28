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
		if err := w.Write([]string{iter.Format(expectedDateFormat), c.names[n]}); err != nil {
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
	if c.startName == "" {
		return errors.New("missing starting name")
	} else {
		found := false
		for _, n := range c.names {
			if c.startName == n {
				found = true
				break
			}
		}
		if !found {
			return errors.New("starting name does not appear in list of user names")
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

	flag.StringVar(&namesInput, "names", "", "comma-separated list of user names to generate rotation from.")
	flag.StringVar(&c.startName, "start-name", "", "the user name to begin the rotation. (defaults to the first provided name in -names list)")
	flag.StringVar(&startDateInput, "start-date", defaultStartDate, "date to begin the rotation from in YYYY-MM-DD format. starts today if left blank.")
	flag.UintVar(&c.days, "days", 365, "number of days to run the rotation for.")
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

	if d, err := time.Parse(expectedDateFormat, startDateInput); err != nil {
		return c, fmt.Errorf("error parsing start date of %q", startDateInput)
	} else {
		c.startDate = d
	}

	if c.startName == "" && len(c.names) > 0 {
		c.startName = c.names[0]
	}

	return c, nil
}
