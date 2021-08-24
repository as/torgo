// This package was created before "flags" had an option for this

package mute

import (
	"os"
	"flag"
)

// Parse silences the noisy flag package Parse() methods
// instead returning errors properly via the error variable
func Parse(f *flag.FlagSet, a []string) error {
	n, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}

	n, os.Stderr = os.Stderr, n
	defer func() {
		n, os.Stderr = os.Stderr, n
	}()

	err = f.Parse(a)
	if err != nil {
		return err
	}

	return nil
}

