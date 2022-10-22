//go:build !windows

package zap

func toOSFilePath(p string) string {
	return p
}
