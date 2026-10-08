package main

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// Platform-specific helpers. Everything that differs between Linux,
// Windows and macOS lives in this file (and in platform_windows.go /
// platform_other.go for process attributes), so the rest of the launcher
// stays platform-neutral.

// angoBinaryName returns the file name of the ango executable on goos.
func angoBinaryName(goos string) string {
	if goos == "windows" {
		return "ango.exe"
	}
	return "ango"
}

// openFolderCommand returns the command that opens dir in the system file
// manager on goos.
func openFolderCommand(goos, dir string) *exec.Cmd {
	switch goos {
	case "windows":
		return exec.Command("explorer", dir)
	case "darwin":
		return exec.Command("open", dir)
	default:
		return exec.Command("xdg-open", dir)
	}
}

// openFolder opens dir in the system file manager without waiting for it.
func openFolder(dir string) error {
	cmd := openFolderCommand(runtime.GOOS, dir)
	hideWindow(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	// explorer.exe exits with status 1 even on success; the exit status is
	// irrelevant here, we only need to reap the process.
	go cmd.Wait()
	return nil
}

var errNoPicker = errors.New("tidak ada pemilih folder (zenity/kdialog) yang tersedia")

// folderPicker describes one way of asking the user for a folder.
type folderPicker struct {
	bin  string
	args []string
}

const pickerTitle = "Pilih folder proyek Ango"

// folderPickers lists the native folder choosers to try on goos, in order.
func folderPickers(goos string) []folderPicker {
	switch goos {
	case "windows":
		script := "Add-Type -AssemblyName System.Windows.Forms; " +
			"$d = New-Object System.Windows.Forms.FolderBrowserDialog; " +
			"$d.Description = '" + pickerTitle + "'; " +
			"if ($d.ShowDialog() -eq 'OK') { [Console]::Out.Write($d.SelectedPath) }"
		return []folderPicker{{"powershell", []string{"-NoProfile", "-STA", "-Command", script}}}
	case "darwin":
		return []folderPicker{{"osascript", []string{"-e",
			`POSIX path of (choose folder with prompt "` + pickerTitle + `")`}}}
	default:
		return []folderPicker{
			{"zenity", []string{"--file-selection", "--directory", "--title=" + pickerTitle}},
			{"kdialog", []string{"--getexistingdirectory", ".", "--title", pickerTitle}},
		}
	}
}

// pickFolder opens a native folder chooser. An empty path with a nil error
// means the user cancelled.
func pickFolder() (string, error) {
	for _, c := range folderPickers(runtime.GOOS) {
		bin, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		cmd := exec.Command(bin, c.args...)
		hideWindow(cmd)
		out, err := cmd.Output()
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", nil // cancelled
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	return "", errNoPicker
}
