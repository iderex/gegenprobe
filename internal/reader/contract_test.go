package reader

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/fixture"
	"github.com/iderex/gegenprobe/internal/model"
)

// The condition the issue asks for by name. The contract runs against every
// reader in the registry, and the set it covered is compared with the set that
// is registered, so a reader cannot be skipped by a filter somebody adds to the
// loop below.
//
// No reader is registered in this build, so this suite covers none and says so
// rather than reporting a pass. What is judged today is the contract itself, in
// the tests under this one.
func TestTheContractRunsAgainstEveryRegisteredReader(t *testing.T) {
	registered := Registrations()

	covered := map[string]bool{}
	for _, registration := range registered {
		participant := registration.Reader.Participant()
		covered[participant] = true
		t.Run(participant, func(t *testing.T) {
			recorded := map[Case][]byte{}
			for _, c := range Cases() {
				name, ok := registration.Fixtures[c]
				if !ok {
					continue
				}
				f, err := fixture.Load(filepath.Join("testdata", name))
				if err != nil {
					t.Fatalf("the %s fixture did not load: %v", c, err)
				}
				recorded[c] = f.Bytes
			}
			for _, v := range Check(registration.Reader, recorded) {
				t.Errorf("%s does not keep the contract: %s", participant, v)
			}
		})
	}

	if len(covered) != len(registered) {
		t.Errorf("%d reader(s) are registered and the contract covered %d of them", len(registered), len(covered))
	}
	if len(registered) == 0 {
		t.Skip("no reader is registered in this build, so the contract judged none of them; the readers are #45 to #49 and each joins this suite by an entry in Registrations")
	}
}

// Every requirement has a double that trips it. A requirement nothing can be
// shown to fail is a requirement nobody has checked the wording of, and this is
// the test that refuses one being added without its near miss.
func TestEveryRequirementIsTrippedByADouble(t *testing.T) {
	tripped := map[Requirement]bool{}
	for _, d := range doubles(t) {
		for _, v := range Check(d.reader, d.recorded) {
			tripped[v.Requirement] = true
		}
	}

	for _, r := range Requirements() {
		if !tripped[r] {
			t.Errorf("no double trips %s, so nothing shows that requirement refusing anything", r)
		}
	}
}

// Each double breaks one requirement and is otherwise the reader that keeps the
// contract, so what the contract reports is the set of requirements broken and
// no other. A double that trips two requirements proves that the contract fires
// and not what it fires on.
func TestEachDoubleTripsExactlyTheRequirementItBreaks(t *testing.T) {
	for _, d := range doubles(t) {
		t.Run(d.name, func(t *testing.T) {
			var got []string
			seen := map[Requirement]bool{}
			for _, v := range Check(d.reader, d.recorded) {
				t.Logf("%s", v)
				if !seen[v.Requirement] {
					seen[v.Requirement] = true
					got = append(got, string(v.Requirement))
				}
			}
			sort.Strings(got)

			want := []string{string(d.breaks)}
			if strings.Join(got, ", ") != strings.Join(want, ", ") {
				t.Errorf("this double breaks %s and the contract reported [%s]", d.breaks, strings.Join(got, ", "))
			}
		})
	}
}

// The reader the doubles are made from keeps the contract, which is what makes
// each of them a one-change neighbour of a reader that passes rather than a
// broken thing beside another broken thing.
func TestTheReaderTheDoublesAreMadeFromKeepsTheContract(t *testing.T) {
	violations := Check(demonstration{}, load(t, demonstrationFixtures))
	for _, v := range violations {
		t.Errorf("the demonstration reader does not keep the contract: %s", v)
	}
}

// A requirement with no statement is one a failure reports as an identifier and
// nothing else, which sends whoever met it to read this package instead of the
// message.
func TestEveryRequirementSaysWhatItRequires(t *testing.T) {
	for _, r := range Requirements() {
		if strings.TrimSpace(r.Statement()) == "" {
			t.Errorf("%s says nothing about what it requires", r)
		}
	}
}

// double is one reader that breaks one requirement, and the recorded files it is
// judged against.
type double struct {
	name     string
	reader   Reader
	recorded map[Case][]byte
	breaks   Requirement
}

func doubles(t *testing.T) []double {
	t.Helper()
	full := load(t, demonstrationFixtures)

	short := map[Case][]byte{}
	for c, b := range full {
		if c == Truncated {
			continue
		}
		short[c] = b
	}

	reads := 0
	return []double{
		{
			name:     "it hands on what it managed to parse out of a truncated file",
			reader:   partialOnTruncation{},
			recorded: full,
			breaks:   RequirementTruncation,
		},
		{
			name:     "it writes a zero into the field the code did not print",
			reader:   zeroForAbsent{},
			recorded: full,
			breaks:   RequirementAbsence,
		},
		{
			name:     "it tidies the code's own label on the way in",
			reader:   normalisingLabels{},
			recorded: full,
			breaks:   RequirementLabel,
		},
		{
			name:     "it drops the count of digits the code wrote",
			reader:   losingSignificance{},
			recorded: full,
			breaks:   RequirementQuantity,
		},
		{
			name:     "its output depends on how often it has been called",
			reader:   unstable{reads: &reads},
			recorded: full,
			breaks:   RequirementStability,
		},
		{
			name:     "it parses another code's file into levels",
			reader:   parsingForeign{},
			recorded: full,
			breaks:   RequirementForeign,
		},
		{
			name:     "it is registered without the file that decides truncation",
			reader:   demonstration{},
			recorded: short,
			breaks:   RequirementDecidable,
		},
	}
}

// demonstrationFixtures is the four recorded files the demonstration reader is
// judged against. They are hand written in its own layout, which is what a
// reader's own entry in Registrations carries.
var demonstrationFixtures = map[Case]string{
	WellFormed:  "a-file-the-reader-is-written-for.fixture",
	Truncated:   "a-file-that-stops-mid-table.fixture",
	FieldAbsent: "a-field-the-code-did-not-print.fixture",
	ForeignCode: "a-file-another-code-wrote.fixture",
}

func load(t *testing.T, names map[Case]string) map[Case][]byte {
	t.Helper()
	out := map[Case][]byte{}
	for c, name := range names {
		f, err := fixture.Load(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("the %s fixture did not load: %v", c, err)
		}
		out[c] = f.Bytes
	}
	return out
}

// The demonstration format. It is a fixed column layout in the shape the codes
// in this bench write, small enough to read in a diff, and it belongs to no code
// in the bench so that nothing here can be mistaken for a reader somebody should
// be using.
const (
	demonstrationHeader = "DEMO LEVELS 1"
	demonstrationEnd    = "END"
	recordWidth         = 40
)

// demonstration is a reader that keeps the contract. It exists so that the
// contract is judged against something, and so that each double below differs
// from a passing reader by one change.
type demonstration struct{}

func (demonstration) Participant() string { return "demonstration" }

func (demonstration) Read(recorded []byte) (Tables, error) {
	lines := strings.Split(string(recorded), "\n")
	if lines[0] != demonstrationHeader {
		return Tables{}, &Foreign{
			Detail: fmt.Sprintf("the first line is %q and this reader is written for %q", lines[0], demonstrationHeader),
		}
	}

	var levels []model.Level
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == demonstrationEnd {
			return Tables{Levels: levels}, nil
		}
		if line == "" && i == len(lines)-1 {
			break
		}
		if len(line) < recordWidth {
			return Tables{}, &Incomplete{
				Line:   i + 1,
				Detail: fmt.Sprintf("a level record is %d characters wide and this one holds %d", recordWidth, len(line)),
			}
		}
		level, err := demonstrationLevel(line, i+1)
		if err != nil {
			return Tables{}, err
		}
		levels = append(levels, level)
	}

	return Tables{}, &Incomplete{
		Line:   len(lines),
		Detail: "the table carries no " + demonstrationEnd + " marker, so where the code meant it to stop is not in the file",
	}
}

func demonstrationLevel(line string, number int) (model.Level, error) {
	index, err := strconv.Atoi(strings.TrimSpace(line[0:5]))
	if err != nil {
		return model.Level{}, fmt.Errorf("line %d: the index column holds %q", number, line[0:5])
	}
	twoJ, err := strconv.Atoi(strings.TrimSpace(line[6:9]))
	if err != nil {
		return model.Level{}, fmt.Errorf("line %d: the twice-J column holds %q", number, line[6:9])
	}
	parity := model.Parity(strings.TrimSpace(line[10:14]))
	if parity != model.Even && parity != model.Odd {
		return model.Level{}, fmt.Errorf("line %d: the parity column holds %q", number, line[10:14])
	}

	energy, err := demonstrationEnergy(strings.TrimSpace(line[28:40]), number)
	if err != nil {
		return model.Level{}, err
	}
	notPrinted, err := model.Absent(model.Declined, model.NotInOutput)
	if err != nil {
		return model.Level{}, err
	}
	notAsked, err := model.Absent(model.NotRequested, model.QuantityNotRequested)
	if err != nil {
		return model.Level{}, err
	}

	return model.Level{
		ID:           strconv.Itoa(index),
		Index:        index,
		Label:        strings.TrimRight(line[15:27], " "),
		TwoJ:         twoJ,
		Parity:       parity,
		MixingWeight: notPrinted,
		Energy:       energy,
		TotalEnergy:  notPrinted,
		Observed:     model.Observed{Energy: notAsked, Uncertainty: notAsked},
	}, nil
}

func demonstrationEnergy(printed string, number int) (model.Quantity, error) {
	if printed == "" {
		return model.Absent(model.Declined, model.NotInOutput)
	}
	value, err := strconv.ParseFloat(printed, 64)
	if err != nil {
		return model.Quantity{}, fmt.Errorf("line %d: the energy column holds %q", number, printed)
	}
	return model.Quantity{
		State:        model.Measured,
		Printed:      printed,
		PrintedUnit:  model.ReciprocalCentimetre,
		Value:        value,
		Unit:         model.ReciprocalCentimetre,
		Significance: significance(printed),
		Marker:       model.Counted,
	}, nil
}

// significance counts the digits the code actually wrote, which for this layout
// is every digit after the leading zeros. It is the crude count and it is enough
// here: what the rule is in general, including the trailing zero that may or may
// not be significant, is #50 and belongs in one place rather than in each
func significance(printed string) int {
	digits, leading := 0, true
	for _, r := range printed {
		switch {
		case r == '0' && leading:
		case r >= '0' && r <= '9':
			leading = false
			digits++
		}
	}
	return digits
}

// normalisingLabels tidies the code's own label, which is the change that makes
// two codes look like they agree about a level neither of them named the same
// way.
type normalisingLabels struct{ demonstration }

func (d normalisingLabels) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	for i := range tables.Levels {
		tables.Levels[i].Label = strings.ToUpper(tables.Levels[i].Label)
	}
	return tables, err
}

// partialOnTruncation hands on the records it managed to parse before the file
// stopped, which is the shape a caller reads whether or not it checks the error.
type partialOnTruncation struct{ demonstration }

func (d partialOnTruncation) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	var incomplete *Incomplete
	if errors.As(err, &incomplete) {
		return Tables{Levels: []model.Level{{ID: "1"}}}, nil
	}
	return tables, err
}

// zeroForAbsent writes a zero into the cell the code left blank. The printed
// text it writes is one that is in the file, so that this double differs from a
// passing reader in the state of one cell and in nothing else.
type zeroForAbsent struct{ demonstration }

func (d zeroForAbsent) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	for i := range tables.Levels {
		if tables.Levels[i].Energy.State == model.Measured {
			continue
		}
		tables.Levels[i].Energy = model.Quantity{
			State:        model.Measured,
			Printed:      "0",
			PrintedUnit:  model.ReciprocalCentimetre,
			Unit:         model.ReciprocalCentimetre,
			Significance: 1,
			Marker:       model.Counted,
		}
	}
	return tables, err
}

// losingSignificance keeps the number and drops the count of digits behind it,
// which is the state in which a value gets printed to more digits than the code
// produced.
type losingSignificance struct{ demonstration }

func (d losingSignificance) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	for i := range tables.Levels {
		if tables.Levels[i].Energy.State != model.Measured {
			continue
		}
		tables.Levels[i].Energy.Significance = 0
	}
	return tables, err
}

// unstable depends on how often it has been called. The identity it moves is not
// one any other requirement reads, so what this double breaks is the stability
// and nothing beside it.
type unstable struct {
	demonstration
	reads *int
}

func (d unstable) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	*d.reads++
	for i := range tables.Levels {
		tables.Levels[i].ID = fmt.Sprintf("%s-%d", tables.Levels[i].ID, *d.reads)
	}
	return tables, err
}

// parsingForeign reads another code's file into levels, which is the failure
// that arrives as a result rather than as an error.
type parsingForeign struct{ demonstration }

func (d parsingForeign) Read(recorded []byte) (Tables, error) {
	tables, err := d.demonstration.Read(recorded)
	var foreign *Foreign
	if errors.As(err, &foreign) {
		return Tables{Levels: []model.Level{{ID: "1"}}}, nil
	}
	return tables, err
}
