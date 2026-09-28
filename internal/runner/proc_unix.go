//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// isolate puts the program in its own process group
func isolate(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// interrupt politely asks the program's whole process group to exit.
func interrupt(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
}

// kill forcefully ends the program's whole process group.
func kill(cmd *exec.Cmd) {
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
