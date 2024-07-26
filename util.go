package main

import (
	"os/user"
	"runtime"
)

func isLinuxRoot() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	return currentUser.Uid == "0"
}
