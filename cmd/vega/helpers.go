package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	dirPerms = 0700
)

// DefaultVegaDir returns the location to vega config files and data files:
//	binary is in /usr/bin/ -> look for /etc/vega/config.toml
//	binary is in /usr/local/vega/bin/ -> look for /usr/local/vega/etc/config.toml
//	binary is in /usr/local/bin/ -> look for /usr/local/etc/vega/config.toml
//	otherwise, look for $HOME/.vega/config.toml
func DefaultVegaDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	if exPath == "/usr/bin" {
		return "/etc/vega"
	}
	if exPath == "/usr/local/vega/bin" {
		return "/usr/local/vega/etc"
	}
	if exPath == "/usr/local/bin" {
		return "/usr/local/etc/vega"
	}
	return os.ExpandEnv("$HOME/.vega")
}

func EnsureDir(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Mkdir(path, dirPerms)
		}
		return err
	}
	return nil
}

type NotFound struct {
	path string
}

func (err *NotFound) Error() string {
	return fmt.Sprintf("not found: %s", err.path)
}

// Exists returns whether a link exists at a given filesystem path.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, &NotFound{path}
	}
	return false, err
}

func waitsig() {
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	sig := <-gracefulStop
	fmt.Printf("caught sig: %+v\n", sig)
}
