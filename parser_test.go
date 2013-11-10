package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	// "testing/quick"
)

func clearRepositories() {
	for key := range repositories {
		delete(repositories, key)
	}
}

var simpleGitHookMessageCreateBranch = "{\"ref\":\"refs/heads/sprintA/taskA\",\"after\":\"0000\",\"before\":\"0000\",\"base_ref\":null,\"created\":true,\"deleted\":false,\"forced\":false,\"repository\":{\"id\":1,\"name\":\"some_repo\",\"master_branch\":\"master\",\"organization\":\"Hello-Org\"},\"head_commit\":{\"id\":\"abc\",\"message\":\"test commit\",\"timestamp\":\"2013-11-09 02:25:48 -0500\"},\"commits\":[{\"id\":\"abc\",\"message\":\"test commit\",\"timestamp\":\"2013-11-09 02:25:48 -0500\"}]}"

func TestGetRestOfRef(t *testing.T) {
	rest := stripRefsHeads(`refs/heads/sprintA/taskA`)
	if rest != "sprintA/taskA" {
		t.Fail()
	}
}

func TestRefRegex(t *testing.T) {
	ref := "refs/heads/sprintA/branchA"
	rest := stripRefsHeads(ref)
	sprintName, branchName, err := extractSprintNameAndBranchName(rest)
	if (sprintName != "sprintA" && branchName != "branchA") || err != nil {
		t.Fail()
	}
}

func TestGUIMergeMessageRegex(t *testing.T) {
	message := "Merge pull request #1 from Hello-Org/sprintA/taskA\n\nadding a new file"
	branchName, err := extractBranchFromMergeMessage(message)

	if branchName != "sprintA/taskA" || err != nil {
		t.Fail()
	}
}

func TestGUIMergeSprintAndBranchFromMessageRegex(t *testing.T) {
	message := "Merge pull request #1 from Hello-Org/sprintA/taskA\n\nadding a new file"
	branchName, err := extractBranchFromMergeMessage(message)
	sprintName, taskName, err2 := extractSprintNameAndBranchName(branchName)

	if err != nil || err2 != nil || (sprintName != "sprintA" && taskName != "taskA") {
		t.Fail()
	}
}

func TestExtractGitHookCreate(t *testing.T) {
	hook, err := extractGithook(simpleGitHookMessageCreateBranch)
	if err != nil {
		t.Fail()
	}

	if hook.Ref != "refs/heads/sprintA/taskA" && hook.Repository.Name != "some_repo" {
		t.Fail()
	}
}

func TestParseGitHookCreate(t *testing.T) {
	hook, err := extractGithook(simpleGitHookMessageCreateBranch)
	if err != nil {
		fmt.Println(err.Error())
		t.Fail()
	}

	applyGitHook(hook)
	ref := stripRefsHeads(hook.Ref)
	sprintName, branchName, err := extractSprintNameAndBranchName(ref)
	if _, ok := repositories[hook.Repository.Id].Branches[branchName]; !ok || branchName != "taskA" || sprintName != "sprintA" {
		fmt.Println(err.Error())
		t.Fail()
	}
}

// given a sequence of githooks, apply them in order
func replayGitHooks(paths ...string) error {
	clearRepositories()
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("no such file or directory: %s\n", path)
			return err
		}

		fmt.Println("applying", path)
		payload, _ := ioutil.ReadFile(path)
		hook, err := extractGithook(string(payload))
		if err != nil {
			fmt.Println(err)
			return err
		}

		err = applyGitHook(hook)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

func TestSimpleCreateAddCommitMergeCLI(t *testing.T) {
	err := replayGitHooks(
		"test-playthroughs/vanilla/created-branch-added-file-commit-changed-file-commit.json",
		"test-playthroughs/vanilla/merge-master-with-foo.json")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	branch, ok := repositories[12720200].Branches["taskA"]
	if !ok {
		fmt.Errorf("could not find branch")
		t.FailNow()
	}

	if !branch.Merged {
		fmt.Errorf("'taskA' isn't merged like it should be")
		t.FailNow()
	}

	if len(branch.Commits) != 2 {
		fmt.Errorf("there should be 2 commits in the taskA branch")
		t.FailNow()
	}
}
