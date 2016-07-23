package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/richterrettich/changelog/domain"
)

func main() {

	var from = flag.String("from", "HEAD", "startpoint for changelog generation")
	var to = flag.String("to", "", "endpoint for changelog generation")
	var dir = flag.String("dir", ".", "the directory the git repository is located")
	flag.Parse()
	groupedCommits := groupConsumer(objectPipe(source(*dir, *from, *to)))
	printMarkdown(groupedCommits)

}

func source(dir, from, to string) <-chan string {
	var cmd *exec.Cmd
	if to == "" {
		cmd = exec.Command("git", "-C", dir, "log", `--pretty=format:hash: %H%n-----%nauthor: %an <%ae>%n-----%nrawSubject: %s%n-----%nrawBody: %b%n-----%nEND-COMMIT%n`, from)
	} else {
		cmd = exec.Command("git", "-C", dir, "log", `--pretty=format:hash: %H%n-----%nauthor: %an <%ae>%n-----%nrawSubject: %s%n-----%nrawBody: %b%n-----%nEND-COMMIT%n`, to+".."+from)
	}

	outputChannel := make(chan string)

	output, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			outputChannel <- scanner.Text()
		}
		defer output.Close()
		close(outputChannel)
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()
	return outputChannel
}

func objectPipe(input <-chan string) <-chan *domain.Commit {
	outputChannel := make(chan *domain.Commit)
	go func() {
		chunk := ""
		result := &domain.Commit{}
		for line := range input {
			if line == "-----" {
				processChunk(chunk, result)
				chunk = ""
			} else {
				chunk = chunk + "\n" + line
			}
			if line == "END-COMMIT" {
				result.ParseBody()
				result.ParseSubject()
				outputChannel <- result
				result = &domain.Commit{}
			}
		}
		close(outputChannel)
	}()
	return outputChannel
}

func processChunk(chunk string, obj *domain.Commit) {
	chunk = strings.TrimSpace(chunk)
	parts := strings.Split(chunk, ":")
	key, value := parts[0], strings.Join(parts[1:], ":")
	switch key {
	case "hash":
		obj.Hash = value
	case "rawSubject":
		obj.RawSubject = value
	case "author":
		obj.Author = value
	case "rawBody":
		obj.RawBody = value
	}
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
