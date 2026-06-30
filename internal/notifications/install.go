package notifications

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type installChannel string

const (
	installChannelScript   installChannel = "script"
	installChannelHomebrew installChannel = "homebrew"
	installChannelGo       installChannel = "go-install"
)

func (n notifier) updateCommand() string {
	switch n.installChannel() {
	case installChannelHomebrew:
		return "brew upgrade sloctl"
	case installChannelGo:
		return "go install github.com/nobl9/sloctl/cmd/sloctl@latest"
	case installChannelScript:
		return n.scriptUpdateCommand()
	default:
		return ""
	}
}

func (n notifier) installChannel() installChannel {
	executablePath, err := os.Executable()
	if err != nil {
		return installChannelScript
	}
	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		resolvedPath = executablePath
	}
	if isHomebrewExecutable(resolvedPath) {
		return installChannelHomebrew
	}
	if isGoInstallExecutable(resolvedPath) {
		return installChannelGo
	}
	return installChannelScript
}

func (n notifier) scriptUpdateCommand() string {
	const scriptURL = "https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash"
	if _, err := exec.LookPath("curl"); err == nil {
		return "curl -fsSL " + scriptURL + " | bash"
	}
	if _, err := exec.LookPath("wget"); err == nil {
		return "wget -O - -q " + scriptURL + " | bash"
	}
	return ""
}

func isHomebrewExecutable(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/Cellar/sloctl/")
}

func isGoInstallExecutable(path string) bool {
	path = filepath.Clean(path)
	for _, binDir := range goBinDirs() {
		if path == filepath.Clean(filepath.Join(binDir, "sloctl")) ||
			path == filepath.Clean(filepath.Join(binDir, "sloctl.exe")) {
			return true
		}
	}
	return false
}

func goBinDirs() []string {
	if goBin := strings.TrimSpace(os.Getenv("GOBIN")); goBin != "" {
		return []string{goBin}
	}
	goPath := strings.TrimSpace(os.Getenv("GOPATH"))
	if goPath == "" {
		homeDir := strings.TrimSpace(os.Getenv("HOME"))
		if homeDir == "" {
			homeDir = strings.TrimSpace(os.Getenv("USERPROFILE"))
		}
		if homeDir == "" {
			return nil
		}
		goPath = filepath.Join(homeDir, "go")
	}
	goPaths := filepath.SplitList(goPath)
	binDirs := make([]string, 0, len(goPaths))
	for _, path := range goPaths {
		if path != "" {
			binDirs = append(binDirs, filepath.Join(path, "bin"))
		}
	}
	return binDirs
}
