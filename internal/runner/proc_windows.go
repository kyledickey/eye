//go:build windows

package runner

import "os/exec"

// isolate does nothing on windows, which has no process groups to lean on.
func isolate(cmd *exec.Cmd) {}

// interrupt ends the program. Windows can't deliver a polite ctrl-c to a
// child process, so this is the same as kill.
func interrupt(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

// kill forcefully ends the program.
func kill(cmd *exec.Cmd) {
	cmd.Process.Kill()
}
