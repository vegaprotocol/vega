package main

import(
	"io/ioutil"

	"github.com/hashicorp/go-multierror"
)

func readFiles(filesWithPath []string) ([][]byte, error) {
	var n = len(filesWithPath)
	contents := make([][]byte, n)
	var errors *multierror.Error
	var err error

	for i := 0; i < n; i++ {
		contents[i], err = ioutil.ReadFile(filesWithPath[i])
		if err != nil {
			multierror.Append(errors, err)
		}
	}

	return contents, errors.ErrorOrNil()
}