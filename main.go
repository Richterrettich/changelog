package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/richterrettich/git-changelog/domain"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

func main() {

	var from = flag.String("from", "HEAD", "startpoint for changelog generation")
	var to = flag.String("to", "", "endpoint for changelog generation")
	var dir = flag.String("dir", ".", "the directory the git repository is located")
	flag.Parse()
	groupedCommits := groupConsumer(source(*dir, *from, *to))
	printMarkdown(groupedCommits)

}

func source(dir, from, to string) <-chan *domain.Commit {

	outputChannel := make(chan *domain.Commit)

	go func() {

		repo, err := git.PlainOpen(dir)

		tagsIter, err := repo.Tags()

		if err != nil {
			panic(err)
		}

		firstTag, err := tagsIter.Next()

		if err != nil {
			panic(err)
		}

		commits, err := repo.Log(&git.LogOptions{})
		if err != nil {
			panic(err)
		}

		commits.ForEach(func(o *object.Commit) error {
			if o.ID() == firstTag.Hash() {
				return storer.ErrStop
			}

			parts := strings.Split(o.Message, "\n")
			subject := parts[0]
			body := ""
			if len(parts) > 1 {
				body = strings.Join(parts[1:len(parts)], "\n")
			}
			result := &domain.Commit{
				RawSubject:      subject,
				RawBody:         body,
				Author:          o.Author.Email,
				Hash:            o.ID().String(),
				Solves:          make([]string, 0),
				BreakingChanges: make([]string, 0),
				Context:         make([]string, 0),
				Errors:          make([]error, 0),
			}

			result.ParseSubject()
			result.ParseBody()

			outputChannel <- result
			return nil
		})

	}()
	return outputChannel
}

func groupConsumer(input <-chan *domain.Commit) map[domain.CommitType][]*domain.Commit {
	result := make(map[domain.CommitType][]*domain.Commit)
	for commit := range input {
		if result[commit.Type] == nil {
			result[commit.Type] = make([]*domain.Commit, 0)
		}
		result[commit.Type] = append(result[commit.Type], commit)
	}
	return result
}

func printMarkdown(input map[domain.CommitType][]*domain.Commit) {
	breakingChanges := make([]string, 0)

	for k, v := range input {
		switch k {
		case domain.Fix:
			fmt.Println("### Bug Fixes")
			printCommits(v)
		case domain.Feature:
			fmt.Println("### Features")
			printCommits(v)

		case domain.Refactoring:
			fmt.Println("### Refactoring")
			printCommits(v)
		}
		for _, commit := range v {
			breakingChanges = append(breakingChanges, commit.BreakingChanges...)
		}
	}
	if len(breakingChanges) > 0 {
		fmt.Println("### BREAKING CHANGES")
		fmt.Println()
		for _, v := range breakingChanges {
			fmt.Println("- " + v)
		}
	}
}

func printCommits(commits []*domain.Commit) {
	for _, commit := range commits {
		fmt.Println("- " + commit.Subject)
		fmt.Println(commit.Body)
		fmt.Println()
	}
}
