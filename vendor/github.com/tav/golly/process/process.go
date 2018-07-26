// Public Domain (-) 2010-2014 The Golly Authors.
// See the Golly UNLICENSE file for details.

// Package process provides utilities to manage the current runtime process.
package process

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const OS = runtime.GOOS

var (
	DisableDefaultExit = false
	SignalHandlers     = make(map[os.Signal][]func())
)

// TODO(tav): It's possible for DisableDefaultExit and SignalHandlers to be
// modified whilst this is running.
func handleSignals() {
	notifier := make(chan os.Signal, 100)
	signal.Notify(notifier)
	var sig os.Signal
	for {
		sig = <-notifier
		handlers, found := SignalHandlers[sig]
		if found {
			for _, handler := range handlers {
				handler()
			}
		}
		if !DisableDefaultExit {
			if sig == syscall.SIGTERM || sig == os.Interrupt {
				os.Exit(1)
			}
		}
	}
}

func prepend(xs []func(), handler func()) []func() {
	return append([]func(){handler}, xs...)
}

func Exit(code int) {
	for _, handler := range SignalHandlers[os.Interrupt] {
		handler()
	}
	os.Exit(code)
}

func SetExitHandler(handler func()) {
	SignalHandlers[syscall.SIGTERM] = prepend(SignalHandlers[syscall.SIGTERM], handler)
	SignalHandlers[os.Interrupt] = prepend(SignalHandlers[os.Interrupt], handler)
}

func SetSignalHandler(signal os.Signal, handler func()) {
	SignalHandlers[signal] = prepend(SignalHandlers[signal], handler)
}

func CreatePidFile(path string) error {
	pidFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	fmt.Fprintf(pidFile, "%d", os.Getpid())
	return pidFile.Close()
}

type ProcessLock struct {
	link     string
	file     string
	acquired bool
}

func Lock(directory, name string) (lock *ProcessLock, err error) {
	file := filepath.Join(directory, fmt.Sprintf("%s-%d.lock", name, os.Getpid()))
	lockFile, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	lockFile.Close()
	link := filepath.Join(directory, name+".lock")
	err = os.Link(file, link)
	if err == nil {
		lock = &ProcessLock{
			link: link,
			file: file,
		}
		SetExitHandler(func() { lock.Release() })
	} else {
		os.Remove(file)
	}
	return
}

func (lock *ProcessLock) Release() {
	os.Remove(lock.file)
	os.Remove(lock.link)
}

// Init acquires a process lock and writes the PID file for the current
// process.
func Init(runPath, name string) error {

	// Get the runtime lock to ensure we only have one process of any given name
	// running within the same run path at any time.
	_, err := Lock(runPath, name)
	if err != nil {
		return fmt.Errorf("Couldn't successfully acquire a process lock:\n\n\t%s\n", err)
	}

	// Write the process ID into a file for use by external scripts.
	return CreatePidFile(filepath.Join(runPath, name+".pid"))

}

// GetIP tries to determine the IP address of the current machine.
func GetIP() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return "", err
	}
	var ip string
	for _, addr := range addrs {
		if strings.Contains(addr, ":") || strings.HasPrefix(addr, "127.") {
			continue
		}
		ip = addr
		break
	}
	if ip == "" {
		return "", fmt.Errorf("Couldn't determine the local IP address")
	}
	return ip, nil
}

// GetAddr returns host:port and fills in empty host parameter with the current
// machine's IP address if need be.
func GetAddr(host string, port int) (string, error) {
	var err error
	if host == "" {
		host, err = GetIP()
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s:%d", host, port), nil
}

// GetAddrListener tries to determine the IP address of the machine when the
// host variable is empty and binds a TCP listener to the given host:port.
func GetAddrListener(host string, port int) (string, net.Listener, error) {
	addr, err := GetAddr(host, port)
	if err != nil {
		return "", nil, err
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, fmt.Errorf("Cannot listen on %s: %v", addr, err)
	}
	return addr, listener, nil
}

func init() {
	go handleSignals()
}
