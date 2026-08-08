// Command externallinks reads every external link in this tree's Markdown and
// reports the ones that no longer answer.
//
// It is deliberately not a leg of go run ./gate. Every other check in this
// repository is a function of the tree and gives the same verdict on a laptop,
// on a runner and in a year. This one asks somebody else's server, so it fails
// for reasons that are not about the change in front of it: a site down for an
// hour, a network that refuses outbound traffic, a host that answers a browser
// and refuses anything else. A pull request refused for any of those is a check
// that gets switched off.
//
// A dead link is still worth knowing about. This repository's documents point at
// the codes it compares, at their licence texts and at the guidance behind its
// own rules, and a reader who cannot follow one of those has lost the evidence
// behind a claim. So it runs on a schedule instead, and the workflow around it
// raises an issue rather than reddening anything.
package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

func main() {
	root := flag.String("root", ".", "the tree to read")
	timeout := flag.Duration("timeout", 20*time.Second, "how long one request may take")
	flag.Parse()

	found, err := collect(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "externallinks:", err)
		os.Exit(2)
	}
	if err := report(os.Stdout, found, fetcher(*timeout)); err != nil {
		fmt.Fprintln(os.Stderr, "externallinks:", err)
		os.Exit(1)
	}
}

var (
	// externalLink takes the target out of [text](target) and out of
	// [label]: target, where the target carries a scheme.
	externalLink = regexp.MustCompile(`\[[^\]]*\]\(\s*<?(https?://[^)>\s]+)>?(?:\s+"[^"]*")?\s*\)|^ {0,3}\[[^\]]+\]:\s*<?(https?://\S+?)>?$`)

	// bare matches a URL written as prose rather than as a link, which is how
	// most of the references in this tree's decision records are written.
	bare = regexp.MustCompile(`https?://[^\s)>\]"'` + "`" + `]+`)

	// fence opens and closes a block whose contents are quoted rather than
	// written, and a URL inside one is an example rather than a reference.
	fence = regexp.MustCompile("^\\s*(```|~~~)")
)

// collect returns every external URL in the tree's Markdown, each with the
// places it is written, sorted so that a run's output does not move for a reason
// that is not about the links.
func collect(root string) (map[string][]string, error) {
	found := map[string][]string{}

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		inside := false
		for i, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
			if fence.MatchString(line) {
				inside = !inside
				continue
			}
			if inside {
				continue
			}
			for _, url := range urlsIn(line) {
				where := fmt.Sprintf("%s:%d", filepath.ToSlash(rel), i+1)
				found[url] = append(found[url], where)
			}
		}
		return nil
	})
	return found, err
}

// urlsIn returns the external URLs written on one line, with the trailing
// punctuation a sentence leaves on the end of one taken off.
func urlsIn(line string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(url string) {
		url = strings.TrimRight(url, ".,;:!?")
		if url == "" || seen[url] {
			return
		}
		seen[url] = true
		out = append(out, url)
	}
	for _, m := range externalLink.FindAllStringSubmatch(line, -1) {
		add(m[1] + m[2])
	}
	for _, m := range bare.FindAllString(line, -1) {
		add(m)
	}
	return out
}

// answer is what one URL said, or the reason nothing was heard from it.
type answer struct {
	status int
	err    error
}

// report asks about every URL and writes what came back. It returns an error
// where at least one did not answer, so the caller's exit status carries the
// same statement as the text.
func report(w io.Writer, found map[string][]string, ask func(string) answer) error {
	urls := make([]string, 0, len(found))
	for url := range found {
		urls = append(urls, url)
	}
	sort.Strings(urls)

	var bad []string
	for _, url := range urls {
		a := ask(url)
		switch {
		case a.err != nil:
			bad = append(bad, fmt.Sprintf("%s\n    %s\n    written at %s", url, a.err, strings.Join(found[url], ", ")))
		case a.status >= 400:
			bad = append(bad, fmt.Sprintf("%s\n    answered %d\n    written at %s", url, a.status, strings.Join(found[url], ", ")))
		}
	}

	fmt.Fprintf(w, "%d external link(s) read, %d of them did not answer.\n", len(urls), len(bad))
	if len(bad) == 0 {
		return nil
	}
	fmt.Fprintln(w)
	for _, b := range bad {
		fmt.Fprintln(w, b)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "A link here is evidence behind a claim in a document. One that does not answer")
	fmt.Fprintln(w, "is either a page that moved, which is an edit, or a site that is down, which is")
	fmt.Fprintln(w, "worth knowing and is not worth refusing a change over.")
	return fmt.Errorf("%d of %d external link(s) did not answer", len(bad), len(urls))
}

// fetcher asks the network. A HEAD is enough for most servers and cheaper for
// all of them; a server that refuses one is asked again with a GET, because
// answering HEAD is optional and a refusal of it says nothing about the page.
func fetcher(timeout time.Duration) func(string) answer {
	client := &http.Client{Timeout: timeout}
	return func(url string) answer {
		if status, err := request(client, http.MethodHead, url); err == nil && status < 400 {
			return answer{status: status}
		}
		status, err := request(client, http.MethodGet, url)
		if err != nil {
			return answer{err: err}
		}
		return answer{status: status}
	}
}

func request(client *http.Client, method, url string) (int, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, err
	}
	// Some of the hosts this tree points at answer a browser and refuse a
	// client with no user agent at all, which would be reported as a dead link
	// on a page anybody can open.
	req.Header.Set("User-Agent", "gegenprobe-externallinks/1 (+https://github.com/iderex/gegenprobe)")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<12))
	return resp.StatusCode, nil
}
