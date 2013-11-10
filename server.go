/*
  this program parses json returned from git hooks and tries
  to figure out which branches are merged and which are new
*/

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// if the process gets SIGUSR1, print the repos
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1)
	go printRepositories(sigChan)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		io.WriteString(res, "hello world")
	})

	http.HandleFunc("/git-hook", func(res http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			io.WriteString(res, "thanks")
		} else {
			fmt.Println("got request", req.Body)
			hook, err := extractGithook(req.FormValue("payload"))
			if err != nil {
				fmt.Println(err.Error())
				return
			}

			applyGitHook(hook)
		}
	})

	fmt.Println(os.Getpid(), "listening on :8080")
	http.ListenAndServe(":8080", nil)
}
