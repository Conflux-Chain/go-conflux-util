package common

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// WriteFileAtomically create a temporary hidden file first
// then move it into place. TempFile assigns mode 0600.
func WriteFileAtomically(file string, content []byte) error {
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return errors.WithMessage(err, "failed to create tmp file")
	}

	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return errors.WithMessage(err, "failed to write tmp file")
	}
	f.Close()

	return errors.WithMessage(os.Rename(f.Name(), file), "failed to rename tmp file")
}
