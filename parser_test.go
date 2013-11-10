package main

import (
	"fmt"
	"testing"
	// "testing/quick"
)

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
