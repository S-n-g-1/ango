//go:build !windows

package main

import "os/exec"

// hideWindow is only meaningful on Windows.
func hideWindow(*exec.Cmd) {}
