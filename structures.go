package main

import (
	"fmt"
)

type GitHookMessage struct {
	Ref, After, Before, Base_ref string
	Created, Deleted, Forced     bool
	Repository                   *Repository
	Commits                      []*Commit
}

type Commit struct {
	Id        string
	Message   string
	Timestamp string
}

type Branch struct {
	Ref     string
	Merged  bool
	Commits []*Commit
}

type Repository struct {
	Id            int
	Name          string
	Master_branch string
	Branches      map[string]*Branch
	Organization  string
}

func (commit Commit) String() string {
	return "\t" + fmt.Sprintf("Commit {id: %s message: %q timestamp: %q}", commit.Id[:5], commit.Message, commit.Timestamp)
}

func printCommits(commits []*Commit) {
	for _, commit := range commits {
		fmt.Println(commit)
	}
}

func (branch Branch) String() string {
	var head = fmt.Sprintf("Branch {ref: %s merged: %t}", branch.Ref, branch.Merged)
	for i := range branch.Commits {
		head += "\n\t" + branch.Commits[len(branch.Commits)-i-1].String()
	}
	return head + "\n"
}

func (repo Repository) String() string {
	var head = fmt.Sprintf("Repository {id:%d name:%s org:%s}", repo.Id, repo.Name, repo.Organization)
	for _, branch := range repo.Branches {
		head += "\n\t" + branch.String()
	}
	return head + "\n"
}
