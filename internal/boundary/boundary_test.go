package boundary

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iderex/gegenprobe/internal/fixture"
)

const module = "github.com/iderex/gegenprobe"

// declaration is what docs/package-boundaries.md says at the commit these
// fixtures were recorded at, written here rather than read from there. A test
// reading that file would assert the file and the parser at once, and the leg is
// what reads the real document: it fails where the real document stops parsing
// and where the real graph stops matching it.
const declaration = `
## .

Decision: 0001
May-import: internal/cli
May-import-in-tests: nothing

## internal/cli

Decision: 0001
May-import: internal/version
May-import-in-tests: nothing

## internal/version

Decision: 0001
May-import: nothing
May-import-in-tests: nothing

## internal/fixture

Decision: 0009
May-import: nothing
May-import-in-tests: nothing

## internal/boundary

Decision: 0009
May-import: nothing
May-import-in-tests: internal/fixture

## internal/commit

Decision: 0009
May-import: nothing
May-import-in-tests: internal/fixture

## gate

Decision: 0009
May-import: internal/boundary, internal/commit, internal/fixture
May-import-in-tests: internal/fixture

## tools/commithygiene

Decision: 0001
May-import: internal/commit
May-import-in-tests: nothing

## tools/decisionindex

Decision: 0000
May-import: nothing
May-import-in-tests: nothing

## tools/externallinks

Decision: 0001
May-import: nothing
May-import-in-tests: nothing
`

func declared(t *testing.T) map[string]Entry {
	t.Helper()
	d, err := ParseDeclaration(declaration)
	if err != nil {
		t.Fatalf("ParseDeclaration: %v", err)
	}
	return d
}

// graph reads one recorded listing. The three fixtures are stored encoded
// because two of them differ from the third by a single import inside a comma
// separated field, which is not something a diff of raw files shows usefully and
// is exactly what a checkout is free to reflow.
func graph(t *testing.T, name string) []Package {
	t.Helper()
	f, err := fixture.Load(filepath.Join("testdata", name+".fixture"))
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	g, err := ParseGraph(string(f.Bytes), module)
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	return g
}

func TestTheGraphAsItIsConforms(t *testing.T) {
	if v := Conform(graph(t, "the-graph-as-it-is"), declared(t)); len(v) != 0 {
		t.Fatalf("the recorded graph produced %d violation(s): %v", len(v), v)
	}
}

func TestAnEdgeTheDeclarationDoesNotPermitIsRefused(t *testing.T) {
	v := Conform(graph(t, "one-edge-that-runs-backwards"), declared(t))
	if len(v) != 1 {
		t.Fatalf("want exactly one violation, got %d: %v", len(v), v)
	}
	if v[0].Package != "internal/version" {
		t.Errorf("the violation names %q rather than the package that made the edge", v[0].Package)
	}
	if !strings.Contains(v[0].Detail, "internal/cli") {
		t.Errorf("the violation does not name the edge: %q", v[0].Detail)
	}
	if !strings.Contains(v[0].Detail, nothing) {
		t.Errorf("the violation does not say what the entry permits instead: %q", v[0].Detail)
	}
}

// The two fixtures above differ in that one import and in nothing else, so the
// one that fails could have passed.
func TestTheOffendingGraphIsOneEdgeFromTheOneThatConforms(t *testing.T) {
	conforming := graph(t, "the-graph-as-it-is")
	offending := graph(t, "one-edge-that-runs-backwards")
	if len(conforming) != len(offending) {
		t.Fatalf("the two graphs hold different packages: %d and %d", len(conforming), len(offending))
	}
	differences := 0
	for i := range conforming {
		if strings.Join(conforming[i].Imports, ",") != strings.Join(offending[i].Imports, ",") ||
			strings.Join(conforming[i].TestImports, ",") != strings.Join(offending[i].TestImports, ",") {
			differences++
		}
	}
	if differences != 1 {
		t.Fatalf("the two graphs differ in %d package(s) rather than one", differences)
	}
}

// Permitting the edge turns the offending graph green, which is what says the
// fixture is proving the declaration rather than something else about the bytes.
func TestPermittingTheEdgeTurnsTheOffendingGraphGreen(t *testing.T) {
	widened := declared(t)
	entry := widened["internal/version"]
	entry.May = []string{"internal/cli"}
	entry.declaredAll = map[string]bool{"internal/cli": true}
	widened["internal/version"] = entry

	if v := Conform(graph(t, "one-edge-that-runs-backwards"), widened); len(v) != 0 {
		t.Fatalf("with the edge permitted the graph still produced %d violation(s): %v", len(v), v)
	}
}

func TestAPackageTheDeclarationDoesNotNameIsRefused(t *testing.T) {
	v := Conform(graph(t, "a-package-the-declaration-does-not-name"), declared(t))
	if len(v) != 1 {
		t.Fatalf("want exactly one violation, got %d: %v", len(v), v)
	}
	if v[0].Package != "internal/somewhere" {
		t.Errorf("the violation names %q rather than the package nothing places", v[0].Package)
	}
	if !strings.Contains(v[0].Detail, "not in the declaration") {
		t.Errorf("the violation does not say what is missing: %q", v[0].Detail)
	}
}

// The register fails closed in both directions: an entry for a package that is
// not in the tree permits edges nothing can make, which is how a declaration
// goes stale after a rename without anything going red.
func TestAnEntryForAPackageThatIsNotInTheTreeIsRefused(t *testing.T) {
	v := Conform(graph(t, "the-graph-as-it-is"), declaredWith(t, `
## internal/gone

Decision: 0001
May-import: nothing
May-import-in-tests: nothing
`))
	if len(v) != 1 {
		t.Fatalf("want exactly one violation, got %d: %v", len(v), v)
	}
	if v[0].Package != "internal/gone" {
		t.Errorf("the violation names %q rather than the entry with nothing behind it", v[0].Package)
	}
}

func declaredWith(t *testing.T, extra string) map[string]Entry {
	t.Helper()
	d, err := ParseDeclaration(declaration + extra)
	if err != nil {
		t.Fatalf("ParseDeclaration: %v", err)
	}
	return d
}

func TestATestImportOutsideTheDeclarationIsRefusedSeparatelyFromASourceImport(t *testing.T) {
	v := Conform(withTestEdge(t, "internal/version", "internal/cli"), declared(t))
	if len(v) != 1 {
		t.Fatalf("want one violation, got %d: %v", len(v), v)
	}
	if !strings.Contains(v[0].Detail, "has a test importing") {
		t.Errorf("a test edge is reported as a source edge: %q", v[0].Detail)
	}
}

// What a package's own source may import, its tests may import too. The two
// lists add rather than replace, or every entry would have to repeat itself.
func TestWhatTheSourceMayImportTheTestsMayImportToo(t *testing.T) {
	if v := Conform(withTestEdge(t, "gate", "internal/commit"), declared(t)); len(v) != 0 {
		t.Fatalf("a test edge the source is permitted was refused: %v", v)
	}
}

// withTestEdge is the recorded graph with one test edge added to one package.
// Building a graph from nothing instead would leave every other entry in the
// declaration with no package behind it, which the register refuses in its own
// right and would drown the thing being asserted.
func withTestEdge(t *testing.T, pkg, imported string) []Package {
	t.Helper()
	g := graph(t, "the-graph-as-it-is")
	for i := range g {
		if g[i].Path == pkg {
			g[i].TestImports = append(g[i].TestImports, imported)
			return g
		}
	}
	t.Fatalf("the recorded graph holds no package %s", pkg)
	return nil
}

func TestParseDeclarationReadsTheDecisionAndBothLists(t *testing.T) {
	d := declared(t)
	if len(d) != 10 {
		t.Fatalf("want ten entries, got %d", len(d))
	}
	for path, e := range d {
		if e.Decision == "" {
			t.Errorf("the entry for %s names no decision", path)
		}
	}
	if got := d["internal/commit"].MayInTests; len(got) != 1 || got[0] != "internal/fixture" {
		t.Errorf("the test list for internal/commit is %v", got)
	}
	if got := d["internal/version"].May; len(got) != 0 {
		t.Errorf("nothing was read as %v rather than as an empty list", got)
	}
}

// A heading whose section carries no decision is prose. Without that the
// document could not explain itself in sections above and below the entries.
func TestAHeadingWithNoDecisionIsNotAnEntry(t *testing.T) {
	d, err := ParseDeclaration(declaration + "\n## Whatever\n\nA paragraph about something.\n")
	if err != nil {
		t.Fatalf("ParseDeclaration: %v", err)
	}
	if _, found := d["Whatever"]; found {
		t.Fatal("a prose heading was read as an entry")
	}
}

func TestAnEntryMissingAListIsRefused(t *testing.T) {
	if _, err := ParseDeclaration("## gate\n\nDecision: 0009\nMay-import: nothing\n"); err == nil {
		t.Fatal("an entry with only one of the two lists parsed")
	}
}

func TestTwoEntriesForOnePackageAreRefused(t *testing.T) {
	one := "## gate\n\nDecision: 0009\nMay-import: nothing\nMay-import-in-tests: nothing\n"
	if _, err := ParseDeclaration(one + "\n" + one); err == nil {
		t.Fatal("a document holding the same package twice parsed")
	}
}

func TestADeclarationWithNoEntryIsRefused(t *testing.T) {
	if _, err := ParseDeclaration("# a document that places nothing\n"); err == nil {
		t.Fatal("a declaration holding no entry parsed")
	}
}

func TestParseGraphKeepsOnlyTheEdgesInsideThisModule(t *testing.T) {
	g, err := ParseGraph(fmt.Sprintf("%s/internal/cli|fmt,%s/internal/version,strings||\n", module, module), module)
	if err != nil {
		t.Fatalf("ParseGraph: %v", err)
	}
	if len(g) != 1 {
		t.Fatalf("want one package, got %d", len(g))
	}
	if len(g[0].Imports) != 1 || g[0].Imports[0] != "internal/version" {
		t.Fatalf("the standard library was not dropped: %v", g[0].Imports)
	}
}

func TestParseGraphNamesTheRootPackageAsADot(t *testing.T) {
	g, err := ParseGraph(fmt.Sprintf("%s|%s/internal/cli|||\n", module, module), module)
	if err == nil {
		t.Fatalf("a line with five fields parsed: %v", g)
	}
	g, err = ParseGraph(fmt.Sprintf("%s|%s/internal/cli||\n", module, module), module)
	if err != nil {
		t.Fatalf("ParseGraph: %v", err)
	}
	if g[0].Path != "." {
		t.Fatalf("the root package is named %q", g[0].Path)
	}
}

func TestParseGraphFoldsTheExternalTestPackageIntoTheTestImports(t *testing.T) {
	g, err := ParseGraph(fmt.Sprintf("%s/gate||%s/internal/commit|%s/internal/fixture\n", module, module, module), module)
	if err != nil {
		t.Fatalf("ParseGraph: %v", err)
	}
	if len(g[0].TestImports) != 2 {
		t.Fatalf("want both test edges, got %v", g[0].TestImports)
	}
}

func TestParseGraphRefusesAnEmptyListing(t *testing.T) {
	if _, err := ParseGraph("\n", module); err == nil {
		t.Fatal("an empty listing parsed as a graph")
	}
}

func TestAViolationPrintsThePackageAndTheDetail(t *testing.T) {
	v := Violation{Package: "internal/version", Detail: "imports something"}
	if v.String() != "internal/version: imports something" {
		t.Fatalf("the printed violation is %q", v.String())
	}
}

// Where the entry permits something, the failure says what, so a reader repairs
// against the list rather than going to look it up.
func TestTheFailureNamesWhatTheEntryPermitsInstead(t *testing.T) {
	g := graph(t, "the-graph-as-it-is")
	for i := range g {
		if g[i].Path == "tools/commithygiene" {
			g[i].Imports = append(g[i].Imports, "internal/version")
		}
	}
	v := Conform(g, declared(t))
	if len(v) != 1 {
		t.Fatalf("want one violation, got %d: %v", len(v), v)
	}
	if !strings.Contains(v[0].Detail, "it permits internal/commit") {
		t.Errorf("the failure does not name the permitted list: %q", v[0].Detail)
	}
}

func TestAListWrittenAsAnEmptyFieldIsAnEmptyList(t *testing.T) {
	d, err := ParseDeclaration("## gate\n\nDecision: 0009\nMay-import: ,\nMay-import-in-tests: nothing\n")
	if err != nil {
		t.Fatalf("ParseDeclaration: %v", err)
	}
	if got := d["gate"].May; len(got) != 0 {
		t.Fatalf("want an empty list, got %v", got)
	}
}
