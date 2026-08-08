package main

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Record 0009 makes headless and unelevated testing a birth requirement of the
// gate tier. This leg is what keeps it true, and it reads source rather than
// watching a run, so what it can prove is bounded: it refuses the shapes that
// have actually caused this failure elsewhere and it cannot prove a test does
// none of them at run time. The stronger version, running the tier where the
// capabilities are absent, is #83, and the failure message says so.
//
// The subject is the test sources of the gate tier. A file carrying the
// integration or regression build tag is another tier and is not read. A file
// under a testdata directory is not read either, because the go tool does not
// compile one, so it is not a test in any tier. Source that is not a test is not
// read at all: the harness starts containers for a living, so the same import
// that is a defect in a test is the point of the command.
const (
	itemExec        = "exec"
	itemNetwork     = "network"
	itemOutside     = "outside"
	itemDisplay     = "display"
	itemDirectory   = "directory"
	itemAbsolute    = "absolute-path"
	itemEnvironment = "environment"
	itemSyscall     = "syscall"
)

// items is the refusable list, in the order record 0009 states it. Every entry
// has a fixture in the suite that trips it and no other.
func items() []string {
	return []string{itemExec, itemNetwork, itemOutside, itemDisplay, itemDirectory, itemAbsolute, itemEnvironment, itemSyscall}
}

var (
	// networkImports is the record's list. net/url is deliberately absent: it
	// parses text and opens nothing.
	networkImports = map[string]bool{
		"net": true, "net/http": true, "net/rpc": true, "net/smtp": true,
		"net/http/httptest": true, "crypto/tls": true,
	}
	displayVariables = map[string]bool{
		"DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true, "BROWSER": true,
	}
	// directoryCalls reach a directory the test did not receive from the
	// framework. t.TempDir is what a gate test writes in and a relative testdata
	// path is what it reads from.
	directoryCalls = map[string]bool{
		"UserHomeDir": true, "UserCacheDir": true, "UserConfigDir": true,
		"TempDir": true, "MkdirTemp": true, "Getwd": true, "Chdir": true,
	}
	environmentCalls = map[string]bool{"Setenv": true, "Unsetenv": true, "Clearenv": true}
	readEnvCalls     = map[string]bool{"Getenv": true, "LookupEnv": true}

	// absolutePath matches a POSIX absolute path and both Windows spellings,
	// because the suite runs on more than one platform and a path of the test's
	// own choosing is the failure this covers.
	absolutePath = regexp.MustCompile(`^(/|[A-Za-z]:[\\/])`)
)

type violation struct {
	file string
	line int
	item string
	why  string
}

// capabilitiesLeg refuses a gate tier test that reaches for a capability the
// tier is not allowed to have.
func capabilitiesLeg(root string) outcome {
	module, err := modulePath(root)
	if err != nil {
		return fail(err.Error())
	}

	var found []violation
	read := 0

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		vs, scanned, err := scanTestSource(filepath.ToSlash(path), src, module, enabled(items()))
		if err != nil {
			return err
		}
		if !scanned {
			return nil
		}
		read++
		found = append(found, vs...)
		return nil
	})
	if err != nil {
		return fail(err.Error())
	}

	if len(found) > 0 {
		lines := make([]string, 0, len(found))
		for _, v := range found {
			lines = append(lines, fmt.Sprintf("%s:%d: %s (%s)", v.file, v.line, v.why, v.item))
		}
		return fail(strings.Join(lines, "\n") +
			"\n\nA test that needs one of these belongs to the integration harness, under its" +
			"\nown build tag, and record 0009 is where widening the list would have to be" +
			"\nargued. Reading source cannot prove a test does none of them at run time, so" +
			"\nthis leg is a floor rather than a guarantee; the executed half is #83.")
	}
	if read == 0 {
		return skip("there is no gate tier test source in this tree, so nothing was read")
	}
	return pass()
}

// scanTestSource returns what one test file reaches for. The second result says
// whether the file was read at all, which is how the caller tells a file it read
// and found nothing in from a file belonging to another tier, and a leg that
// examined nothing from one that examined everything and passed.
func scanTestSource(path string, src []byte, module string, on map[string]bool) ([]violation, bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", path, err)
	}
	if otherTier(f) {
		return nil, false, nil
	}

	var found []violation
	at := func(pos token.Pos) int { return fset.Position(pos).Line }
	add := func(pos token.Pos, item, why string) {
		if on[item] {
			found = append(found, violation{file: path, line: at(pos), item: item, why: why})
		}
	}

	for _, spec := range f.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		switch {
		case p == "syscall" || p == "golang.org/x/sys" || strings.HasPrefix(p, "golang.org/x/sys/"):
			add(spec.Pos(), itemSyscall, "imports "+p+", which is how a privilege request or a capability change arrives")
		case p == "os/exec":
			add(spec.Pos(), itemExec, "imports os/exec, which is how an elevation tool, a service manager, a firewall utility, a certificate store or a container engine is started")
		case networkImports[p]:
			add(spec.Pos(), itemNetwork, "imports "+p+", which reaches the network, loopback included")
		case outsideTheStandardLibrary(p, module):
			add(spec.Pos(), itemOutside, "imports "+p+", which is outside the standard library and outside this module, so what it opens is not this tree's to know")
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			sel, ok := x.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "os" {
				return true
			}
			name := sel.Sel.Name
			switch {
			case directoryCalls[name]:
				add(sel.Pos(), itemDirectory, "calls os."+name+", which is a directory of the test's own choosing rather than t.TempDir or a relative testdata path")
			case environmentCalls[name]:
				add(sel.Pos(), itemEnvironment, "calls os."+name+", which changes an environment another test reads; t.Setenv restores it")
			case readEnvCalls[name] && len(x.Args) > 0:
				if v, ok := stringLiteral(x.Args[0]); ok && displayVariables[v] {
					add(sel.Pos(), itemDisplay, "reads "+v+", which is a display, and a test that reads it to decide whether to skip is deciding to be conditional")
				}
			}
		case *ast.BasicLit:
			if x.Kind != token.STRING {
				return true
			}
			v, err := strconv.Unquote(x.Value)
			if err != nil || !isAbsolutePathLiteral(v) {
				return true
			}
			add(x.Pos(), itemAbsolute, "holds the absolute path literal "+x.Value)
		}
		return true
	})

	return found, true, nil
}

// isAbsolutePathLiteral narrows record 0009's "string constant beginning with /
// or with a drive letter" by one property no path has, which is a line break.
// A test that holds another file's source as a literal is the ordinary way to
// write a fixture in this tree, and such a literal begins with a build
// constraint comment often enough that without this the check would refuse its
// own suite. The cost is that a path followed by a newline inside one literal is
// not seen, which no path written to be used looks like.
func isAbsolutePathLiteral(v string) bool {
	if strings.ContainsAny(v, "\r\n") {
		return false
	}
	return absolutePath.MatchString(v)
}

// otherTier reports whether a build constraint names the integration or the
// regression tag. A file with no constraint is a gate test, so the strict tier
// is what a contributor gets without deciding anything.
func otherTier(f *ast.File) bool {
	for _, group := range f.Comments {
		for _, c := range group.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			if namesAnotherTier(expr) {
				return true
			}
		}
	}
	return false
}

func namesAnotherTier(e constraint.Expr) bool {
	switch x := e.(type) {
	case *constraint.TagExpr:
		return x.Tag == "integration" || x.Tag == "regression"
	case *constraint.NotExpr:
		return namesAnotherTier(x.X)
	case *constraint.AndExpr:
		return namesAnotherTier(x.X) || namesAnotherTier(x.Y)
	case *constraint.OrExpr:
		return namesAnotherTier(x.X) || namesAnotherTier(x.Y)
	}
	return false
}

// outsideTheStandardLibrary reports whether an import path names neither the
// standard library nor this module. A standard library path has no dot in its
// first segment, which is the rule the toolchain itself uses.
func outsideTheStandardLibrary(path, module string) bool {
	if path == module || strings.HasPrefix(path, module+"/") {
		return false
	}
	first, _, _ := strings.Cut(path, "/")
	return strings.Contains(first, ".")
}

func stringLiteral(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// enabled turns a list of item ids into the set the scan judges by. The leg
// enables all of them; the suite turns one off at a time to show each fixture is
// green without the item it was written for rather than green by construction.
func enabled(list []string) map[string]bool {
	on := make(map[string]bool, len(list))
	for _, item := range list {
		on[item] = true
	}
	return on
}

// modulePath reads the module line out of go.mod, so that this module's own
// packages are not counted as a dependency outside the standard library.
func modulePath(root string) (string, error) {
	body, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("%s names no module", filepath.ToSlash(filepath.Join(root, "go.mod")))
}
