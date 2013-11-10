package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var repositories = make(map[int]*Repository)
var sprints = make(map[string][]*Branch) // sprint "A" has all these branches

func printRepositories(sigChan chan os.Signal) {
	<-sigChan
	for _, repo := range repositories {
		fmt.Println(repo)
	}
	go printRepositories(sigChan)
}

var sprintBranchRegex = regexp.MustCompile(`([-_\(\)\w]+)\/([-_\(\)\w]+)`)
var mergeMessageOrgBranchRegex = regexp.MustCompile(`Merge pull request #\d+ from [-_\(\)\w]+\/(.*?)\n\n`)

func stripRefsHeads(ref string) string {
	return strings.TrimPrefix(ref, `refs/heads/`)
}

func extractSprintNameAndBranchName(ref string) (string, string, error) {
	matches := sprintBranchRegex.FindAllStringSubmatch(ref, -1)
	extractionError := errors.New("could not extract sprint name and branch name from: " + ref)
	if len(matches) != 1 {
		return "", "", extractionError
	}

	if len(matches[0]) != 3 {
		return "", "", extractionError
	}

	return matches[0][1], matches[0][2], nil
}

func extractBranchFromMergeMessage(message string) (string, error) {
	matches := mergeMessageOrgBranchRegex.FindAllStringSubmatch(message, -1)
	extractionError := errors.New("could not extract org and branch name from merge message: " + message)

	if len(matches) != 1 {
		return "", extractionError
	}

	if len(matches[0]) != 2 {
		return "", extractionError
	}

	return matches[0][1], nil
}

func extractGithook(text string) (*GitHookMessage, error) {
	var hook GitHookMessage
	err := json.Unmarshal([]byte(text), &hook)
	if err != nil {
		return nil, errors.New("unable to parse git hook message: " + err.Error())
	}
	return &hook, nil
}

func applyGitHook(hook *GitHookMessage) error {
	var repo, repoExists = repositories[hook.Repository.Id]
	if !repoExists {
		repo = hook.Repository
		repo.Branches = make(map[string]*Branch)
		repositories[repo.Id] = repo
		fmt.Println("created repository", repo.Name, "with id", hook.Repository.Id)
	}

	rest := stripRefsHeads(hook.Ref)
	_, branchName, err := extractSprintNameAndBranchName(rest)
	if err != nil && rest != repo.Master_branch {
		fmt.Println(err.Error())
		return err
	} else if rest == repo.Master_branch {
		branchName = repo.Master_branch
	}

	// is a new branch being created that doesn't exist
	// track branches that follow sprint/task scheme
	branch, branchExists := repo.Branches[branchName]
	if !branchExists {
		if hook.Created {
			// is the branch being created, don't add the commits just yet
			branch = &Branch{branchName, false, nil, repo}
			repo.Branches[branchName] = branch
		} else if branchName != repo.Master_branch && hook.Base_ref == "" {
			// if we arent' tracking a branch from the start and
			// we're not merging into the master branch, don't do anything
			fmt.Println("branch", branchName, "doesn't exist. Ignoring.")
			return nil
		}
	}

	// is the branch being deleted
	if hook.Deleted {
		delete(repo.Branches, branchName)
		return nil
	}

	// force-pushing an existing branch
	if hook.Forced && !hook.Created && branchExists {
		var foundPreviousCommit = false

		for i, commit := range branch.Commits {
			// replace this commit and everything after it with the new commits
			if commit.Id == hook.Before {
				branch.Commits = append(branch.Commits[:i-1], hook.Commits...)
				foundPreviousCommit = true
				break
			}
		}

		if !foundPreviousCommit {
			branch.Commits = append(branch.Commits, hook.Commits...)
		}
	}

	// merge
	if branchName == repo.Master_branch {
		var lastCommit = hook.Commits[len(hook.Commits)-1]
		if hook.Base_ref == "" && strings.HasPrefix(lastCommit.Message, "Merge pull request #") {
			branchName, _ := extractBranchFromMergeMessage(lastCommit.Message)
			mergedBranch, branchExists := repo.Branches[branchName]
			if !branchExists {
				// these might not be all of the commits but this is the best that we can get
				repo.Branches[branchName] = &Branch{branchName, true, hook.Commits, repo}
			} else {
				mergedBranch.Merged = true
			}
		} else if hook.Base_ref != "" {
			// merge from command line
			rest := stripRefsHeads(hook.Base_ref)
			_, baseBranchName, err := extractSprintNameAndBranchName(rest)
			if err != nil {
				return err
			}

			repo.Branches[baseBranchName].Merged = true
		} else {
			fmt.Println("I don't know how this was merged")
		}
	} else {
		// normal commit
		if branch.Commits == nil {
			branch.Commits = hook.Commits
		} else {
			branch.Commits = append(branch.Commits, hook.Commits...)
		}
	}

	return nil
}
