package domain

import (
	"errors"
	"regexp"
	"strings"
)

type History []*Commit

type Commit struct {
	RawSubject      string
	RawBody         string
	Subject         string
	Body            string
	Author          string
	Hash            string
	Solves          []string
	BreakingChanges []string
	Type            CommitType
	Context         []string
	Errors          []error
}

type CommitType int

const (
	Fix CommitType = iota
	Feature
	Clean
	Chore
	Refactoring
	Build
	Test
	Unknown
)

func (c *Commit) ParseSubject() {
	if len(strings.Split(c.RawSubject, "\n")) > 1 {
		c.Errors = append(c.Errors, errors.New("The Subject should not contain newlines"))
	}
	if len(c.RawSubject) > 100 {
		c.Errors = append(c.Errors, errors.New("The Subject should have more than 100 characters"))
	}
	parts := strings.Split(c.RawSubject, ":")

	rawType, rest := parts[0], parts[1:]
	if len(rest) == 0 {
		c.Type = Unknown
		c.Subject = strings.TrimSpace(rawType)
		c.Errors = append(c.Errors, errors.New("Subject does not follow the format TYPE[(context)]: SUBJECT"))
		return
	}

	if strings.Contains(rawType, "(") {
		parts = strings.Split(rawType, "(")
		rawType = parts[0]
		if !strings.Contains(parts[1], ")") {
			c.Errors = append(c.Errors, errors.New("Missing ) in context list"))
		}
		c.Context = strings.Split(strings.TrimRight(parts[1], ")"), ",")
	}

	rawType = strings.TrimSpace(rawType)
	switch strings.ToLower(rawType) {
	case "fix":
		c.Type = Fix
	case "feat", "feature":
		c.Type = Feature
	case "refac", "refactoring", "refactor":
		c.Type = Refactoring
	case "clean":
		c.Type = Clean
	case "chore":
		c.Type = Chore
	case "build":
		c.Type = Build
	case "test":
		c.Type = Test
	default:
		c.Type = Unknown
		c.Errors = append(c.Errors, errors.New(rawType+" is an unknown commit type."))
	}

	c.Subject = strings.TrimSpace(strings.Join(rest, ":"))
}

func (c *Commit) ParseBody() {

	if c.RawBody == "" {
		return
	}

	breakingChangesRegex := regexp.MustCompile(`breaking[-_ ]changes:\s*\n`)

	parts := strings.Split(c.RawBody, "\n\n")

	if len(parts) > 3 {
		c.Errors = append(c.Errors, errors.New("Body has too many parts. Expected a maximum of  3."))
	}

	for _, v := range parts {
		normalized := strings.TrimSpace(v)
		if breakingChangesRegex.MatchString(strings.ToLower(normalized)) {
			c.parseBreakingChanges(normalized)
		} else if strings.HasPrefix(strings.ToLower(normalized), "solves:") {
			c.parseSolves(normalized)
		} else {
			c.Body = normalized
		}
	}
}

func (c *Commit) parseBreakingChanges(bs string) {

	regex := regexp.MustCompile(`\n\s+-`)
	parts := regex.Split(bs, -1)
	if len(parts) == 0 {
		c.Errors = append(c.Errors, errors.New("Unable to parse breaking changes."))
		return
	}

	breakingChanges := parts[1:]

	for i, v := range breakingChanges {
		breakingChanges[i] = strings.TrimSpace(v)
	}

	c.BreakingChanges = breakingChanges
}

func (c *Commit) parseSolves(so string) {
	rawSolves := strings.Split(so, ":")[1]

	if strings.TrimSpace(rawSolves) == "" {
		c.Errors = append(c.Errors, errors.New("No resolved Issues detected even though they where declared."))
	}
	solves := strings.Split(rawSolves, ",")
	for i, v := range solves {
		solves[i] = strings.TrimSpace(v)
	}
	c.Solves = solves
}
