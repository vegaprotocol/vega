package memory

import (
	"errors"

	"github.com/pbnjay/memory"
)

func TotalMemory() (uint64, error) {
	mem := memory.TotalMemory()
	if mem == 0 {
		return 0, errors.New("accessible memory size could not be determined")
	}

	return mem, nil
}
