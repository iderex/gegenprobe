// Command decisionindex generates docs/decisions/README.md from the decision
// records beside it, so that the index cannot drift against the directory it
// describes. The format it reads is 0000-decision-records.md.
//
// It refuses to write an index it could not derive: a record it cannot parse is
// an error rather than a line it silently leaves out, because a missing line in
// a generated index is exactly the drift the generator exists to remove.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// recordName matches NNNN-short-slug.md and nothing else, so README.md and any
// other prose in the directory is not mistaken for a record.
var recordName = regexp.MustCompile(`^([0-9]{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$`)

// headingLine matches the first line of a record, "# NNNN. Title".
var headingLine = regexp.MustCompile(`^# ([0-9]{4})\. (.+)$`)

// permittedStatus is the set 0000 fixes. A value outside it is an error here so
// that the generated index never carries a status the format does not define.
var permittedStatus = map[string]bool{
	"proposed":   true,
	"accepted":   true,
	"superseded": true,
}

type record struct {
	file         string
	number       string
	title        string
	status       string
	date         string
	supersededBy string
}

func main() {
	dir := flag.String("dir", filepath.Join("docs", "decisions"), "directory holding the decision records")
	check := flag.Bool("check", false, "write nothing; exit non-zero if the index is not current")
	flag.Parse()

	if err := run(*dir, *check); err != nil {
		fmt.Fprintln(os.Stderr, "decisionindex:", err)
		os.Exit(1)
	}
}

func run(dir string, check bool) error {
	records, err := readRecords(dir)
	if err != nil {
		return err
	}
	want := render(records)
	path := filepath.Join(dir, "README.md")

	got, err := os.ReadFile(path)
	switch {
	case err == nil:
		if bytes.Equal(normalise(got), want) {
			return nil
		}
	case errors.Is(err, os.ErrNotExist):
		// No index yet. Writing one is the whole job; -check reports it.
	default:
		return err
	}

	if check {
		return fmt.Errorf("%s is not current; run: go run ./tools/decisionindex", filepath.ToSlash(path))
	}
	return os.WriteFile(path, want, 0o644)
}

// normalise removes carriage returns so that a working tree checked out with
// CRLF endings compares equal to the LF bytes this command writes. Git stores
// LF either way, so this is about the file on disk and not about the commit.
func normalise(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

func readRecords(dir string) ([]record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var records []record
	seen := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := recordName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		r, err := readRecord(filepath.Join(dir, e.Name()), m[1])
		if err != nil {
			return nil, err
		}
		if other, dup := seen[r.number]; dup {
			return nil, fmt.Errorf("number %s is used by both %s and %s", r.number, other, r.file)
		}
		seen[r.number] = r.file
		records = append(records, r)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("no decision records found under %s", filepath.ToSlash(dir))
	}

	sort.Slice(records, func(i, j int) bool { return records[i].number < records[j].number })
	return records, nil
}

func readRecord(path, numberFromName string) (record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return record{}, err
	}
	name := filepath.Base(path)
	lines := strings.Split(string(normalise(raw)), "\n")

	if len(lines) == 0 {
		return record{}, fmt.Errorf("%s: empty", name)
	}
	h := headingLine.FindStringSubmatch(lines[0])
	if h == nil {
		return record{}, fmt.Errorf("%s: first line is not \"# NNNN. Title\"", name)
	}
	if h[1] != numberFromName {
		return record{}, fmt.Errorf("%s: heading says %s, filename says %s", name, h[1], numberFromName)
	}

	fields, err := headerFields(lines, name)
	if err != nil {
		return record{}, err
	}

	r := record{
		file:         name,
		number:       fields["Number"],
		title:        fields["Title"],
		status:       fields["Status"],
		date:         fields["Date"],
		supersededBy: fields["Superseded-By"],
	}
	for _, want := range []struct{ key, value string }{
		{"Number", r.number}, {"Title", r.title}, {"Status", r.status}, {"Date", r.date},
	} {
		if want.value == "" {
			return record{}, fmt.Errorf("%s: header field %s is missing", name, want.key)
		}
	}
	if r.number != numberFromName {
		return record{}, fmt.Errorf("%s: Number is %s, filename says %s", name, r.number, numberFromName)
	}
	if r.title != h[2] {
		return record{}, fmt.Errorf("%s: heading title and Title field differ", name)
	}
	if !permittedStatus[r.status] {
		return record{}, fmt.Errorf("%s: Status %q is not one of proposed, accepted, superseded", name, r.status)
	}
	if r.status == "superseded" && r.supersededBy == "" {
		return record{}, fmt.Errorf("%s: status is superseded and Superseded-By is missing", name)
	}
	return r, nil
}

// headerFields reads the run of "Field: value" lines that follows the heading
// and its blank line, stopping at the next blank line.
func headerFields(lines []string, name string) (map[string]string, error) {
	i := 1
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	fields := map[string]string{}
	for ; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			break
		}
		key, value, ok := strings.Cut(lines[i], ": ")
		if !ok {
			return nil, fmt.Errorf("%s: header line %q is not \"Field: value\"", name, lines[i])
		}
		fields[key] = strings.TrimSpace(value)
	}
	return fields, nil
}

func render(records []record) []byte {
	var b strings.Builder
	b.WriteString("# Decision records\n\n")
	b.WriteString("One file per decision, in the format fixed by\n")
	b.WriteString("[0000](0000-decision-records.md).\n\n")
	b.WriteString("This index is generated. Edit a record and regenerate it rather than\n")
	b.WriteString("editing this file:\n\n")
	b.WriteString("    go run ./tools/decisionindex\n\n")
	b.WriteString("| Number | Title | Status | Date |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, r := range records {
		status := r.status
		if r.status == "superseded" {
			status = fmt.Sprintf("superseded by %s", r.supersededBy)
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s | %s | %s |\n", r.number, r.file, escapePipes(r.title), status, r.date)
	}
	return []byte(b.String())
}

// escapePipes keeps a title containing a pipe from splitting its table row.
func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
