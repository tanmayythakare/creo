//go:build windows

package process

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// checkProcess returns nil if the process is still alive on Windows
func checkProcess(proc *os.Process) error {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(proc.Pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return err
	}

	// STILL_ACTIVE = 259
	if exitCode == 259 {
		return nil
	}
	return os.ErrProcessDone
}

// IsTerminalProcess checks if a PID belongs to a known console/terminal shell process on Windows
func IsTerminalProcess(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var size uint32 = 1024
	buf := make([]uint16, size)
	err = windows.QueryFullProcessImageName(handle, 0, &buf[0], &size)
	if err != nil {
		return false
	}
	name := windows.UTF16ToString(buf[:size])
	base := strings.ToLower(filepath.Base(name))
	terminalShells := []string{
		"cmd.exe", "powershell.exe", "pwsh.exe", "wt.exe", "wsl.exe", "bash.exe", "conhost.exe", "openconsole.exe",
	}
	for _, shell := range terminalShells {
		if base == shell {
			return true
		}
	}
	return false
}
