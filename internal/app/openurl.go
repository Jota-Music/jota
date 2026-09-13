//go:build !android

package app

import "errors"

func openExternal(url string) error {
	return errors.New("openExternal not supported on this platform")
}
