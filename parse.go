/*
  this program parses json returned from git hooks and tries
  to figure out which branches are merged and which are new
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

var steps = []string{
	"created-branch-added-file-commit-changed-file-commit",
	"sync-fixup-shove",
	"added-1-commit-added-2-commit-added-3-commit",
	"sync-fixup-2-shove",
	"merge-foo-from-github-ui",
}

var repositories = make(map[int]*Repository)

func main() {
	for _, step := range steps {
		text, _ := ioutil.ReadFile(step + ".json")
		parseGitHookMessage(text)
	}

	for _, repo := range repositories {
		fmt.Println(repo)
	}
}

func parseGitHookMessage(text []byte) {
	var hook GitHookMessage
	json.Unmarshal(text, &hook)

	var repo = hook.Repository
	var storedRepo, ok = repositories[repo.Id]
	if !ok {
		repo.Branches = make(map[string]*Branch)
		storedRepo = repo
		repositories[storedRepo.Id] = repo
	}

	if hook.Created {
		if _, ok := storedRepo.Branches[hook.Ref]; !ok {
			storedRepo.Branches[hook.Ref] = &Branch{hook.Ref, false, hook.Commits}
		}
	}

	if hook.Forced && !hook.Created {
		var branch = storedRepo.Branches[hook.Ref]
		var found = false

		for i, commit := range branch.Commits {
			if commit.Id == hook.Before {
				branch.Commits = append(branch.Commits[:i-1], hook.Commits...)
				found = true
				break
			}
		}

		if !found {
			branch.Commits = append(branch.Commits, hook.Commits...)
		}
	}

	if hook.Ref == "refs/heads/"+storedRepo.Master_branch {
		var lastCommit = hook.Commits[len(hook.Commits)-1]
		if hook.Base_ref == "" && strings.HasPrefix(lastCommit.Message, "Merge pull request #") {
			// ugly ugly ugly code to get the branch name for this merge
			indexAfterOrg := strings.Index(lastCommit.Message, storedRepo.Organization+"/") + len(storedRepo.Organization) + 1
			indexBeforeNewlines := strings.Index(lastCommit.Message[indexAfterOrg:], "\n\n")

			branchName := lastCommit.Message[indexAfterOrg : indexAfterOrg+indexBeforeNewlines]
			storedRepo.Branches["refs/heads/"+branchName].Merged = true
		} else if hook.Base_ref != "" {
			storedRepo.Branches[hook.Base_ref].Merged = true
		} else {
			fmt.Println("I don't know how this was merged")
		}
	}
}
