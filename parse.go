/*
  this program parses json returned from git hooks and tries
  to figure out which branches are merged and which are new
*/

package main

import (
	"encoding/json"
	"fmt"
	"io"
	// "io/ioutil"
	"net/http"
	"strings"
)

var repositories = make(map[int]*Repository)

func main() {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "hello world")
	})

	http.HandleFunc("/git-hook", func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			io.WriteString(res, "thanks")
		} else {
			parseGitHookMessage(req.FormValue("payload"))
			req.Body.Close()

			for _, repo := range repositories {
				fmt.Println(repo)
			}
		}
	})

	fmt.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func parseGitHookMessage(text string) {
	var hook GitHookMessage
	err := json.Unmarshal([]byte(text), &hook)
	if err != nil {
		fmt.Println("json unmarshal:", err.Error())
		fmt.Print(string(text))
		return
	}

	var repo = hook.Repository
	var storedRepo, repoExists = repositories[repo.Id]
	if !repoExists {
		repo.Branches = make(map[string]*Branch)
		storedRepo = repo
		repositories[storedRepo.Id] = repo
	}

	if hook.Deleted {
		delete(storedRepo.Branches, hook.Ref)
	}

	if hook.Created {
		if _, ok := storedRepo.Branches[hook.Ref]; !ok {
			storedRepo.Branches[hook.Ref] = &Branch{hook.Ref, false, hook.Commits}
		}
	}

	if hook.Forced && !hook.Created {
		var branch, branchExists = storedRepo.Branches[hook.Ref]
		if branchExists {
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
	}

	if hook.Ref == "refs/heads/"+storedRepo.Master_branch {
		var lastCommit = hook.Commits[len(hook.Commits)-1]
		if hook.Base_ref == "" && strings.HasPrefix(lastCommit.Message, "Merge pull request #") {
			// ugly ugly ugly code to get the branch name for this merge
			indexAfterOrg := strings.Index(lastCommit.Message, storedRepo.Organization+"/") + len(storedRepo.Organization) + 1
			indexBeforeNewlines := strings.Index(lastCommit.Message[indexAfterOrg:], "\n\n")

			branchName := lastCommit.Message[indexAfterOrg : indexAfterOrg+indexBeforeNewlines]
			branch, branchExists := storedRepo.Branches["refs/heads/"+branchName]
			if !branchExists {
				storedRepo.Branches["refs/heads/"+branchName] = &Branch{"refs/heads/" + branchName, true, hook.Commits}
			} else {
				branch.Merged = true
			}
		} else if hook.Base_ref != "" {
			storedRepo.Branches[hook.Base_ref].Merged = true
		} else {
			fmt.Println("I don't know how this was merged")
		}
	} else {
		// normal commit
		var commits = storedRepo.Branches[hook.Ref].Commits

		if commits == nil {
			commits = hook.Commits
		} else {
			storedRepo.Branches[hook.Ref].Commits = append(commits, hook.Commits...)
		}

	}
}
