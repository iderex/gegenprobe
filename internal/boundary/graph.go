package boundary

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// Format is what the toolchain is asked to print, one line per package: the
// import path, the imports of its own source, the imports of its tests, and the
// imports of its external test package. The last of those is a separate field
// and is folded into the test imports here, because a test in package foo_test
// makes exactly the same edge as one in package foo.
const Format = `{{.ImportPath}}|{{join .Imports ","}}|{{join .TestImports ","}}|{{join .XTestImports ","}}`

// Graph asks the toolchain for the module's own import graph. It is the only
// thing here that runs a process, and it is deliberately the whole of it: what
// the graph means is judged by Conform over data, which is what lets a recorded
// graph stand in for a real one in a test.
func Graph(dir string) ([]Package, error) {
	module, err := run(dir, "go", "list", "-m")
	if err != nil {
		return nil, err
	}
	module = strings.TrimSpace(module)
	if module == "" {
		return nil, fmt.Errorf("the toolchain named no module in %s", dir)
	}

	listed, err := run(dir, "go", "list", "-f", Format, "./...")
	if err != nil {
		return nil, err
	}
	return ParseGraph(listed, module)
}

// ParseGraph reads what the toolchain printed into packages, keeping only the
// edges inside this module. An edge to the standard library or to a dependency
// is a question docs/dependencies.md answers, and reading it here would give two
// documents an opinion about the same thing.
func ParseGraph(listed, module string) ([]Package, error) {
	var out []Package

	for _, line := range strings.Split(strings.ReplaceAll(listed, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) != 4 {
			return nil, fmt.Errorf("the line %q holds %d field(s) and not 4", line, len(fields))
		}
		p := Package{Path: relative(fields[0], module)}
		p.Imports = inside(fields[1], module)
		p.TestImports = inside(fields[2]+","+fields[3], module)
		out = append(out, p)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("the toolchain listed no package, so no graph was judged")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// relative turns a full import path into the form the declaration writes, with
// "." for the package at the module root.
func relative(path, module string) string {
	if path == module {
		return "."
	}
	return strings.TrimPrefix(path, module+"/")
}

// inside keeps the edges that stay in this module and drops the rest.
func inside(field, module string) []string {
	var out []string
	for _, p := range strings.Split(field, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == module || strings.HasPrefix(p, module+"/") {
			out = append(out, relative(p, module))
		}
	}
	sort.Strings(out)
	return out
}

// run executes one read only toolchain command and hands back what it printed.
func run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		detail := ""
		if ee, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(ee.Stderr))
		}
		if detail == "" {
			return "", fmt.Errorf("%s %s: %v", name, strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("%s %s: %v: %s", name, strings.Join(args, " "), err, detail)
	}
	return string(out), nil
}
