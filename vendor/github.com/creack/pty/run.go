//go:build !windows
// +build !windows

package pty

import (
	"os"
	"os/exec"
	"syscall"
)

// Start assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
func Start(c *exec.Cmd) (pty *os.File, err error) {
	return StartWithSize(c, nil)
}

// StartWithSize assigns a pseudo-terminal tty os.File to c.Stdin, c.Stdout,
// and c.Stderr, calls c.Start, and returns the File of the tty's
// corresponding pty.
//
// This will resize the pty to the specified size before starting the command.
//
// Updated for Go 1.20+ compatibility: Ctty must reference a valid FD in the
// child process. We pass the tty as an ExtraFiles entry so it gets FD 3 in
// the child, and set Ctty = 3.
func StartWithSize(c *exec.Cmd, sz *Winsize) (pty *os.File, err error) {
	pty, tty, err := Open()
	if err != nil {
		return nil, err
	}
	if sz != nil {
		if err = Setsize(pty, sz); err != nil {
			_ = pty.Close()
			_ = tty.Close()
			return nil, err
		}
	}
	if c.Stdout == nil {
		c.Stdout = tty
	}
	if c.Stderr == nil {
		c.Stderr = tty
	}
	if c.Stdin == nil {
		c.Stdin = tty
	}

	// Pass the tty as an extra file so it has a known FD in the child.
	// ExtraFiles FDs start at 3 (after stdin=0, stdout=1, stderr=2).
	// The child FD = 3 + len(c.ExtraFiles) before appending.
	cttyFd := 3 + len(c.ExtraFiles)
	c.ExtraFiles = append(c.ExtraFiles, tty)

	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.Setctty = true
	c.SysProcAttr.Setsid = true
	c.SysProcAttr.Ctty = cttyFd

	err = c.Start()
	// Close the tty in the parent after Start, regardless of success/failure.
	_ = tty.Close()
	if err != nil {
		_ = pty.Close()
		return nil, err
	}
	return pty, nil
}
