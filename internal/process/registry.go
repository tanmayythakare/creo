package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tanmaythakare/creo/internal/ui"
)

// Registry maps service name → PID
type Registry map[string]int

const registryDir = ".creo"
const registryFile = "pids.json"

func ensureCreoDir(root string) error {
	dir := filepath.Join(root, registryDir)
	fi, err := os.Stat(dir)
	if err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("✗ .creo exists as a file, please delete it")
		}
		return nil
	}
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return err
}

func handleAntivirusLock(path string, operation func() error) error {
	for {
		err := operation()
		if err == nil {
			return nil
		}
		fmt.Println()
		ui.Warn("⚠ Could not write to .creo/ — file may be locked by antivirus.\n  Try adding your project folder to antivirus exclusions.")
		choice := ui.Ask("Retry? [y/N]", "n")
		choice = strings.ToLower(strings.TrimSpace(choice))
		if choice != "y" && choice != "yes" {
			return fmt.Errorf("✗ could not access %s: file locked", path)
		}
	}
}

// LoadRegistry reads .creo/pids.json, returns empty registry if not found or corrupted
func LoadRegistry(root string) (Registry, error) {
	path := filepath.Join(root, registryDir, registryFile)

	var data []byte
	var err error
	for i := 0; i < 3; i++ {
		data, err = os.ReadFile(path)
		if err == nil || os.IsNotExist(err) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		if os.IsNotExist(err) {
			return Registry{}, nil
		}
		return nil, fmt.Errorf("✗ could not access .creo/pids.json: file locked")
	}

	var reg Registry
	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return json.Unmarshal(data, &reg)
	}()

	if err != nil {
		fmt.Println()
		ui.Warn("⚠ pids.json was corrupted, resetting")
		_ = os.Remove(path)
		return Registry{}, nil
	}

	return reg, nil
}

// SaveRegistry writes the registry to .creo/pids.json atomically
func SaveRegistry(root string, reg Registry) error {
	if err := ensureCreoDir(root); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Join(root, registryDir)
	destPath := filepath.Join(dir, registryFile)
	tmpPath := filepath.Join(dir, "pids.tmp")

	writeOp := func() error {
		f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := f.Write(data); err != nil {
			return fmt.Errorf("✗ could not write PID file: disk full")
		}

		if err := f.Sync(); err != nil {
			return fmt.Errorf("✗ could not write PID file: disk full")
		}

		return nil
	}

	err = handleAntivirusLock(tmpPath, writeOp)
	if err != nil {
		return err
	}

	renameOp := func() error {
		return os.Rename(tmpPath, destPath)
	}

	err = handleAntivirusLock(destPath, renameOp)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}

// SetPID adds or updates a service PID in the registry
func SetPID(root, service string, pid int) error {
	if pid <= 0 {
		return nil
	}
	reg, _ := LoadRegistry(root)
	if reg == nil {
		reg = Registry{}
	}
	reg[service] = pid
	return SaveRegistry(root, reg)
}

// RemovePID deletes a service entry from the registry
func RemovePID(root, service string) error {
	reg, err := LoadRegistry(root)
	if err != nil {
		return nil // nothing to remove
	}
	delete(reg, service)
	return SaveRegistry(root, reg)
}

// LockRegistry creates .creo/creo.lock and writes the current PID to it.
// If the lock file exists and the PID inside is alive, it returns an error unless user forces.
func LockRegistry(root string) error {
	if err := ensureCreoDir(root); err != nil {
		return err
	}

	dir := filepath.Join(root, registryDir)
	lockPath := filepath.Join(dir, "creo.lock")
	data, err := os.ReadFile(lockPath)
	if err == nil {
		// A lock file exists, check if the process is alive
		var pid int
		if _, scanErr := fmt.Sscanf(string(data), "%d", &pid); scanErr == nil && pid > 0 {
			if IsPIDAlive(pid) {
				fmt.Println()
				ui.Warn(fmt.Sprintf("⚠ Another instance of creo appears to be running in this project.\n  (.creo/creo.lock exists)\n  If this is stale, remove it and try again."))
				choice := ui.Ask("Force continue? [y/N]", "n")
				choice = strings.ToLower(strings.TrimSpace(choice))
				if choice != "y" && choice != "yes" {
					return fmt.Errorf("✗ creo is already running in this project")
				}
			}
		}
	}

	// Create/overwrite lock file with current PID
	myPID := os.Getpid()
	writeOp := func() error {
		return os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", myPID)), 0644)
	}

	return handleAntivirusLock(lockPath, writeOp)
}

// UnlockRegistry removes the lock file if it was created by the current process
func UnlockRegistry(root string) {
	lockPath := filepath.Join(root, registryDir, "creo.lock")
	data, err := os.ReadFile(lockPath)
	if err == nil {
		var pid int
		if _, scanErr := fmt.Sscanf(string(data), "%d", &pid); scanErr == nil && pid == os.Getpid() {
			_ = os.Remove(lockPath)
		}
	}
}

// IsPIDAlive checks if a PID is alive on the OS
func IsPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return checkProcess(proc) == nil
}
