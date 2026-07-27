package notifications

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type installChannel string

const (
	installChannelUnknown  installChannel = ""
	installChannelScript   installChannel = "script"
	installChannelHomebrew installChannel = "homebrew"
	installChannelGo       installChannel = "go-install"
)

type updateCommand struct {
	display    string
	executable string
	args       []string
}

func (c updateCommand) available() bool {
	return c.executable != ""
}

func detectUpdateCommand() updateCommand {
	switch detectInstallChannel() {
	case installChannelHomebrew:
		return updateCommand{
			display:    "brew upgrade sloctl",
			executable: "brew",
			args:       []string{"upgrade", "sloctl"},
		}
	case installChannelGo:
		return updateCommand{
			display:    "go install github.com/nobl9/sloctl/cmd/sloctl@latest",
			executable: "go",
			args:       []string{"install", "github.com/nobl9/sloctl/cmd/sloctl@latest"},
		}
	case installChannelScript:
		return scriptUpdateCommand()
	default:
		return updateCommand{}
	}
}

func detectInstallChannel() installChannel {
	executablePath, err := os.Executable()
	if err != nil {
		return installChannelUnknown
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
	if isInstallScriptExecutable(resolvedPath) {
		return installChannelScript
	}
	return installChannelUnknown
}

func scriptUpdateCommand() updateCommand {
	const scriptURL = "https://raw.githubusercontent.com/nobl9/sloctl/main/install.bash"
	if _, err := exec.LookPath("curl"); err == nil {
		return bashPipelineUpdateCommand("curl -fsSL " + scriptURL + " | bash")
	}
	if _, err := exec.LookPath("wget"); err == nil {
		return bashPipelineUpdateCommand("wget -O - -q " + scriptURL + " | bash")
	}
	return updateCommand{}
}

func bashPipelineUpdateCommand(display string) updateCommand {
	return updateCommand{
		display:    display,
		executable: "bash",
		args:       []string{"-o", "pipefail", "-c", display},
	}
}

func (n notifier) runCommand(command updateCommand) error {
	//nolint:gosec // The executable and arguments come from fixed sloctl update definitions.
	cmd := exec.Command(command.executable, command.args...)
	cmd.Stdin = n.stdin
	cmd.Stdout = n.stdout
	cmd.Stderr = n.stderr
	return cmd.Run()
}

func isHomebrewExecutable(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/Cellar/sloctl/")
}

func isInstallScriptExecutable(path string) bool {
	path = strings.ReplaceAll(filepath.Clean(path), `\`, "/")
	return path == "/usr/local/bin/sloctl" ||
		strings.HasSuffix(path, "/usr/local/bin/sloctl.exe")
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
