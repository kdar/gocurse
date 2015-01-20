package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gitTagRegex    = regexp.MustCompile(`refs/tags/(.*?)[,\)]`)
	gitBranchRegex = regexp.MustCompile(`refs/heads/(.*?)[,\)]`)
)

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func isGitRepo() bool {
	_, err := runGit("rev-parse", "--git-dir")
	if _, ok := err.(*exec.ExitError); err != nil && ok {
		return false
	}

	return true
}

// func gitFile() (*gocurse.File, error) {
// 	return nil, nil
// }

type gitCommit struct {
	Branch  string
	Tag     string
	Commit  string
	Message string
	Date    time.Time
}

func gitChangelog() (string, error) {
	commitsOut, err := runGit("log", "--decorate=full", `--format=|||%d|||%h|||%ct|||%s|||__END__`)
	if err != nil {
		return "", err
	}

	branch := "master"
	var commits []gitCommit
	lines := strings.Split(commitsOut, "__END__\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|||")
		tag := ""
		commit := parts[2]
		timestamp := parts[3]
		message := parts[4]

		matches := gitTagRegex.FindStringSubmatch(parts[1])
		if matches != nil {
			tag = matches[1]
		}

		matches = gitBranchRegex.FindStringSubmatch(parts[1])
		if matches != nil {
			branch = matches[1]
		}

		date, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return "", err
		}

		commits = append(commits, gitCommit{
			Branch:  branch,
			Tag:     tag,
			Commit:  commit,
			Message: message,
			Date:    time.Unix(date, 0),
		})
	}

	changelog := ""
	lastTag := ""
	for i, commit := range commits {
		if (commit.Tag != "" && lastTag != commit.Tag) || i == 0 {
			t := commit.Tag
			if t == "" {
				t = commit.Branch
			}

			changelog += fmt.Sprintf("\n%s / %s", t, commit.Date.Format("2006/01/02"))
			changelog += "\n=================\n\n"

			lastTag = commit.Tag
		}

		changelog += fmt.Sprintf(" * %s\n", commit.Message)
	}

	return changelog, err
}

func gitLatestTag() (string, error) {
	tag, err := runGit("describe", "--tags", "--abbrev=0")
	if _, ok := err.(*exec.ExitError); err != nil && ok {
		return "", err
	}

	return strings.TrimRight(tag, "\n "), nil
}

func gitArchive(prefix string) (io.Reader, error) {
	archive, err := runGit("archive", "--format=zip", "--prefix="+prefix, "--worktree-attributes", "HEAD")
	if _, ok := err.(*exec.ExitError); err != nil && ok {
		return nil, err
	}

	buf := bytes.NewBufferString(archive)
	return buf, nil
}
