package commit

import (
	"fmt"
	"strings"
)

const (
	fieldSeparator  = "\x1f"
	recordSeparator = "\x00"
)

// Parse reads what git printed under Format into commits. A record with the
// wrong number of fields is an error rather than a skipped line: a parser that
// quietly drops a record it did not understand is a check that examined fewer
// commits than it was given and said nothing about it.
func Parse(stream []byte) ([]Commit, error) {
	text := strings.TrimSuffix(string(stream), recordSeparator)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	var out []Commit
	for i, record := range strings.Split(text, recordSeparator) {
		fields := strings.SplitN(record, fieldSeparator, 4)
		if len(fields) != 4 {
			return nil, fmt.Errorf("record %d holds %d field(s) and not 4: %q", i+1, len(fields), truncate(record))
		}
		out = append(out, Commit{
			SHA:     fields[0],
			Author:  fields[1],
			Parents: len(strings.Fields(fields[2])),
			Message: fields[3],
		})
	}
	return out, nil
}

// truncate keeps an error about a malformed record readable when the record is
// a whole commit message.
func truncate(s string) string {
	const limit = 60
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
