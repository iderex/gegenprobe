package golden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The passing artefact every case below changes one record of: a document with
// prose above a table, in the shape the index over the decision records has.
func table(rows ...string) string {
	return strings.Join(append([]string{
		"# An index",
		"",
		"This part is generated.",
		"",
		"| Number | Title | Status |",
		"| --- | --- | --- |",
	}, rows...), "\n") + "\n"
}

func rows() []string {
	return []string{
		"| [0000](0000-a.md) | The first one | accepted |",
		"| [0001](0001-b.md) | The second one | accepted |",
		"| [0002](0002-c.md) | The third one | accepted |",
	}
}

func TestAnArtefactAgreesWithItself(t *testing.T) {
	if found := Compare([]byte(table(rows()...)), []byte(table(rows()...))); len(found) > 0 {
		t.Fatalf("an artefact was found to differ from itself: %s", Report("x", found))
	}
}

// The whole of what this package is for. One record differs and the report says
// which one, by the name a reader of the table would use for it.
func TestADifferingRecordIsNamedAndNotLocated(t *testing.T) {
	changed := rows()
	changed[1] = "| [0001](0001-b.md) | The second one, retitled | accepted |"

	found := Compare([]byte(table(rows()...)), []byte(table(changed...)))

	if len(found) != 1 {
		t.Fatalf("one changed record produced %d difference(s): %s", len(found), Report("x", found))
	}
	if found[0].Key != "0001" {
		t.Errorf("the difference is keyed %q rather than by the record", found[0].Key)
	}

	report := Report("docs/decisions/README.md", found)
	for _, want := range []string{"docs/decisions/README.md", "record 0001", "The second one", "The second one, retitled"} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not carry %q:\n%s", want, report)
		}
	}
	if strings.Contains(report, "byte") || strings.Contains(report, "offset") {
		t.Errorf("the report locates the difference rather than naming it:\n%s", report)
	}
}

// The two directions are opposite statements and are kept apart. A record that
// was renumbered is one of each, and reporting it as a single change would say
// the wrong thing about both.
func TestARecordDroppedAndARecordAddedAreDifferentFindings(t *testing.T) {
	found := Compare([]byte(table(rows()...)), []byte(table(rows()[0], rows()[1], "| [0003](0003-d.md) | A fourth | accepted |")))

	if len(found) != 2 {
		t.Fatalf("a renumbered record produced %d difference(s): %s", len(found), Report("x", found))
	}
	if found[0].Key != "0002" || !strings.Contains(found[0].Why, "nothing produced it") {
		t.Errorf("the dropped record is reported as %q, %q", found[0].Key, found[0].Why)
	}
	if found[1].Key != "0003" || !strings.Contains(found[1].Why, "does not hold it") {
		t.Errorf("the added record is reported as %q, %q", found[1].Key, found[1].Why)
	}
}

// A checkout decides line endings and a producer does not, so a comparison that
// read one as a change would report every record of every artefact on half the
// machines this runs on.
func TestALineEndingIsNotADifference(t *testing.T) {
	onDisk := strings.ReplaceAll(table(rows()...), "\n", "\r\n")

	if found := Compare([]byte(onDisk), []byte(table(rows()...))); len(found) > 0 {
		t.Fatalf("a carriage return was read as a change to the artefact: %s", Report("x", found))
	}
}

// The key is what a reader calls the row, not the file it points at. A rename
// moves the target of every link at once, and a comparison keyed on the target
// would report the whole table as replaced rather than as retargeted.
func TestARecordKeepsItsKeyWhenItsLinkMoves(t *testing.T) {
	moved := rows()
	moved[0] = "| [0000](records/0000-a.md) | The first one | accepted |"

	found := Compare([]byte(table(rows()...)), []byte(table(moved...)))

	if len(found) != 1 || found[0].Key != "0000" {
		t.Fatalf("a moved link produced %d difference(s), the first keyed %q", len(found), found[0].Key)
	}
	if !strings.Contains(found[0].Why, "disagree") {
		t.Errorf("a retargeted record is reported as %q rather than as a disagreement", found[0].Why)
	}
}

func TestATablesOwnScaffoldingIsNotARecord(t *testing.T) {
	for _, line := range []string{"| --- | --- | --- |", "| :--- | ---: |", "not a table row", "", "  |  |  "} {
		if key := keyOf(line); key != "" {
			t.Errorf("%q was keyed %q, and it names no record", line, key)
		}
	}
	if key := keyOf("| Number | Title | Status |"); key != "Number" {
		t.Errorf("the header row is keyed %q rather than by its first cell", key)
	}
}

// A key that appears twice identifies nothing, so both lines fall back to the
// ordered comparison rather than one of them replacing the other in silence.
func TestARepeatedKeyIdentifiesNothing(t *testing.T) {
	twice := []string{rows()[0], rows()[0]}

	found := Compare([]byte(table(twice...)), []byte(table(rows()[0], rows()[1])))

	if len(found) == 0 {
		t.Fatal("a table holding one key twice compared equal to a different table")
	}
}

func TestTextOutsideTheTableIsReportedByQuotingIt(t *testing.T) {
	changed := strings.Replace(table(rows()...), "This part is generated.", "This part is generated by something else.", 1)

	found := Compare([]byte(table(rows()...)), []byte(changed))

	if len(found) != 1 {
		t.Fatalf("one changed line of prose produced %d difference(s)", len(found))
	}
	if found[0].Key != "" {
		t.Errorf("a line outside the table was given the key %q", found[0].Key)
	}
	report := Report("x", found)
	if !strings.Contains(report, "This part is generated by something else.") {
		t.Errorf("the report does not quote the line that differs:\n%s", report)
	}
}

// A golden that matches passes, and one that does not fails naming the record.
// The failing half is exercised through check rather than through Assert,
// because a helper whose failure path is only reached by a failing test is a
// path nobody reads.
func TestCheckPassesAMatchingGoldenAndRefusesAStaleOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.md")
	if err := os.WriteFile(path, []byte(table(rows()...)), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := check(path, []byte(table(rows()...))); err != nil {
		t.Fatalf("a golden matching its producer was refused: %v", err)
	}

	changed := rows()
	changed[2] = "| [0002](0002-c.md) | The third one, superseded | superseded by 0004 |"
	err := check(path, []byte(table(changed...)))
	if err == nil {
		t.Fatal("a golden that no longer matches its producer passed")
	}
	for _, want := range []string{"record 0002", "index.md", "-update"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q:\n%v", want, err)
		}
	}
}

func TestCheckSaysWhatToDoWhereThereIsNoGoldenYet(t *testing.T) {
	err := check(filepath.Join(t.TempDir(), "absent.md"), []byte(table(rows()...)))

	if err == nil {
		t.Fatal("a golden that does not exist compared equal to something")
	}
	if !strings.Contains(err.Error(), "-update") {
		t.Errorf("the refusal does not say how to write it: %v", err)
	}
}

// The update flag is the whole reason this is a helper rather than a comparison
// somebody writes each time, so what it does is asserted rather than described.
func TestTheUpdateFlagRewritesTheGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.md")
	if err := os.WriteFile(path, []byte(table(rows()[0])), 0o644); err != nil {
		t.Fatal(err)
	}

	*update = true
	t.Cleanup(func() { *update = false })

	Assert(t, path, []byte(table(rows()...)))

	rewritten, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(rewritten) != table(rows()...) {
		t.Errorf("the golden was not rewritten to what the producer wrote:\n%s", rewritten)
	}

	*update = false
	Assert(t, path, []byte(table(rows()...)))
}
