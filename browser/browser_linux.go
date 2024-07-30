// We're not going to support opening the user's browser on Linux because xdg-open doesn't work
// as expected when running as root and there isn't a good way to drop privileges. We favor
// consistency across sudo/non-sudo over having an extra feature for non-sudo users.

package browser

import "fmt"

func OpenURL(url string) error {
	return fmt.Errorf("opening browser is unsupported on linux")
}

func IsSupported() bool {
	return false
}
