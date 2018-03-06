package book

import (
	"fmt"
)

type Party struct {
	Name string
}

func (p *Party) String() string {
	return fmt.Sprintf("%v", p.Name)
}

func (p *Party) GetId() string {
	return p.Name
}
