// +build windows

package main

// SetUlimits is currently a no-op on Windows.
func (l *NodeCommand) SetUlimits() error {
	return nil
}
