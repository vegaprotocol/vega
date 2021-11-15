//go:build windows
// +build windows

package node

// SetUlimits is currently a no-op on Windows.
func (l *NodeCommand) SetUlimits() error {
	return nil
}
