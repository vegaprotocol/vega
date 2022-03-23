package entities

import (
	"time"
)

type NodeID struct{ ID }

func NewNodeID(id string) NodeID {
	return NodeID{ID: ID(id)}
}

type Node struct {
	ID       NodeID
	VegaTime time.Time
}
