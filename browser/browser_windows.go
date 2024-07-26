package browser

import "golang.org/x/sys/windows"

func OpenURL(url string) error {
	return windows.ShellExecute(0, nil, windows.StringToUTF16Ptr(url), nil, nil, windows.SW_SHOWNORMAL)
}

func IsSupported() bool {
	return true
}
