package browser

import (
	"io"
	"os/exec"
)

func OpenURL(url string) error {
	cmd := exec.Command("open", url)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func IsSupported() bool {
	return true
}
