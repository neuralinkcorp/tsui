package clipboard

import "errors"

var errUnavailable = errors.New("couldn't copy: clipboard unavailable")
