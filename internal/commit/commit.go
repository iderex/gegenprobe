// Package commit judges the commit messages a change is made of. It reads
// nothing: the commits are handed to it and the rules are handed to it, so the
// same judgement is available to the command that runs before a push and to the
// job that runs on a pull request without either of them holding a second copy
// of it.
//
// Two things are judged. Whether a subject names the issue the commit belongs
// to, which keeps the link in the one line that survives into every log listing
// and every blame view. And whether the message is written in the characters
// docs/commit-messages.md allows, which is an allowlist because the shape of the
// attack nobody has written up yet is not on any list of forbidden characters.
package commit

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Commit is one commit as git reported it. Parents is the number of them, which
// is how a merge is recognised without parsing a subject the forge wrote.
type Commit struct {
	SHA     string
	Author  string
	Parents int
	Message string
}

// Subject is the first line of the message. A message with no line feed is all
// subject, which is the shape a one line commit takes.
func (c Commit) Subject() string {
	if i := strings.IndexByte(c.Message, '\n'); i >= 0 {
		return c.Message[:i]
	}
	return c.Message
}

// Origin says where the head of the change came from. A contribution from
// outside this repository cannot know an issue number before the issue exists,
// so the subject rule reports rather than fails for one. The character rule is
// not relaxed for either.
type Origin int

const (
	Internal Origin = iota
	External
)

// Rule names which of the two rules a finding came from, so a caller can report
// one and fail on the other without matching on wording.
type Rule int

const (
	SubjectNamesNoIssue Rule = iota
	CharacterNotAllowed
)

// Finding is one thing wrong with one commit. Line is one based and is zero
// where the rule is not about a particular line. Fatal is false where the
// finding is reported rather than refused, which happens only for a subject on a
// contribution from outside.
type Finding struct {
	SHA    string
	Rule   Rule
	Line   int
	Detail string
	Fatal  bool
}

func (f Finding) String() string {
	where := f.SHA
	if f.Line > 0 {
		where += fmt.Sprintf(", line %d", f.Line)
	}
	return where + ": " + f.Detail
}

// Span is one run of code points a message may hold, inclusive at both ends.
type Span struct {
	Lo rune
	Hi rune
}

// Rules is what docs/commit-messages.md declares. It is parsed from that
// document rather than written here, so widening either half is an edit to the
// place the reasoning is, and a reader who finds the constant finds the argument
// beside it.
type Rules struct {
	Allowed []Span
	Bots    []string
}

var (
	allowedLine = regexp.MustCompile(`(?m)^Allowed:\s*U\+([0-9A-F]{4,6})(?:-U\+([0-9A-F]{4,6}))?\s*$`)
	botLine     = regexp.MustCompile(`(?m)^Exempt-Author:\s*(\S.*?)\s*$`)
	issueRef    = regexp.MustCompile(`#[0-9]+`)
)

// ParseRules reads the declaration out of the document. It refuses a document
// declaring nothing, because a rules file that parsed to an empty allowlist
// would refuse every commit and a rules file that parsed to an empty one in the
// other direction would refuse none, and neither failure announces itself.
func ParseRules(doc string) (Rules, error) {
	var r Rules

	for _, m := range allowedLine.FindAllStringSubmatch(doc, -1) {
		lo, err := codePoint(m[1])
		if err != nil {
			return Rules{}, err
		}
		hi := lo
		if m[2] != "" {
			if hi, err = codePoint(m[2]); err != nil {
				return Rules{}, err
			}
		}
		if hi < lo {
			return Rules{}, fmt.Errorf("the declared span U+%s to U+%s runs backwards", m[1], m[2])
		}
		r.Allowed = append(r.Allowed, Span{Lo: lo, Hi: hi})
	}
	for _, m := range botLine.FindAllStringSubmatch(doc, -1) {
		r.Bots = append(r.Bots, m[1])
	}

	if len(r.Allowed) == 0 {
		return Rules{}, errors.New("the declaration holds no allowed span, so nothing says which characters a message may hold")
	}
	return r, nil
}

// codePoint reads one hexadecimal code point out of the declaration. A number
// that is not a code point is refused here rather than silently becoming a span
// nothing can be inside, which would be an allowlist that refuses everything and
// says nothing about why.
func codePoint(hex string) (rune, error) {
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || n > 0x10FFFF {
		return 0, fmt.Errorf("the declaration names U+%s, which is above the highest code point there is", hex)
	}
	return rune(n), nil
}

// permits answers whether one code point is inside the declared spans.
func (r Rules) permits(c rune) bool {
	for _, s := range r.Allowed {
		if c >= s.Lo && c <= s.Hi {
			return true
		}
	}
	return false
}

// exempt answers whether this commit's subject is judged at all. A merge is
// exempt because the forge wrote its subject; a trusted bot is exempt by exact
// author name, because a suffix rule would exempt anybody who chose the name.
func (r Rules) exempt(c Commit) bool {
	if c.Parents > 1 {
		return true
	}
	for _, b := range r.Bots {
		if strings.Contains(c.Author, b) {
			return true
		}
	}
	return false
}

// Judge returns every finding over the commits given, in the order the commits
// were given and, within a commit, subject before characters.
func Judge(commits []Commit, rules Rules, origin Origin) []Finding {
	var out []Finding
	for _, c := range commits {
		if !rules.exempt(c) && !issueRef.MatchString(c.Subject()) {
			out = append(out, Finding{
				SHA:    c.SHA,
				Rule:   SubjectNamesNoIssue,
				Detail: fmt.Sprintf("the subject names no issue: %q", c.Subject()),
				Fatal:  origin == Internal,
			})
		}
		out = append(out, characterFindings(c, rules)...)
	}
	return out
}

// characterFindings walks the message a code point at a time and names the line,
// the code point and what it is, because a message that fails on an invisible
// character is one nobody can repair from a verdict that only says it failed.
func characterFindings(c Commit, rules Rules) []Finding {
	var out []Finding
	line := 1
	for _, r := range c.Message {
		if r == '\n' {
			if !rules.permits(r) {
				out = append(out, Finding{
					SHA:    c.SHA,
					Rule:   CharacterNotAllowed,
					Line:   line,
					Detail: "U+000A LINE FEED is not in the allowlist",
					Fatal:  true,
				})
			}
			line++
			continue
		}
		if rules.permits(r) {
			continue
		}
		out = append(out, Finding{
			SHA:    c.SHA,
			Rule:   CharacterNotAllowed,
			Line:   line,
			Detail: fmt.Sprintf("U+%04X (%s) is not in the allowlist", r, describe(r)),
			Fatal:  true,
		})
	}
	return out
}

// describe says what the code point is without printing it. Printing the
// character is what a reader would ask for and is exactly wrong here: the ones
// worth refusing are the ones that look like something else, and a terminal
// showing a no-break space as a space would report the fault as its own cause.
func describe(r rune) string {
	switch {
	case r == '\t':
		return "a tab"
	case r == '\r':
		return "a carriage return"
	case r < 0x20 || r == 0x7F:
		return "a control character"
	case r == 0x00A0:
		return "a no-break space, which renders as an ordinary space"
	case r < 0x80:
		return "an ASCII character outside the allowed spans"
	default:
		return "outside ASCII"
	}
}

// Failed answers whether the findings refuse the change. A reported finding
// counts as examined and not as passed, which is why the caller prints every
// finding and this only decides the exit status.
func Failed(findings []Finding) bool {
	for _, f := range findings {
		if f.Fatal {
			return true
		}
	}
	return false
}
