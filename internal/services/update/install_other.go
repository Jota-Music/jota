//go:build !linux && !darwin && !windows

package update

func targetDir() (string, error) { return "", ErrUnsupported }

func install(string) error { return ErrUnsupported }
