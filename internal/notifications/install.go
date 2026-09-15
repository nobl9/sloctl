package notifications

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const installationQueryTimeout = 2 * time.Second

type updateCommand struct {
	display    string
	executable string
	args       []string
}

func (c updateCommand) available() bool {
	return c.display != "" && c.executable != ""
}

func detectUpdateCommand() updateCommand {
	executablePath, err := os.Executable()
	if err != nil {
		return updateCommand{}
	}
	resolvedPath, err := filepath.EvalSymlinks(executablePath)
	if err != nil {
		return updateCommand{}
	}

	if command := homebrewUpdateCommand(resolvedPath); command.available() {
		return command
	}
	return goInstallUpdateCommand(resolvedPath)
}

func (n notifier) runCommand(command updateCommand) error {
	//nolint:gosec // The executable and arguments come from fixed sloctl update definitions.
	cmd := exec.Command(command.executable, command.args...)
	cmd.Stdin = n.stdin
	cmd.Stdout = n.stdout
	cmd.Stderr = n.stderr
	return cmd.Run()
}

func homebrewUpdateCommand(sloctlPath string) updateCommand {
	brewExecutable, err := exec.LookPath("brew")
	if err != nil {
		return updateCommand{}
	}
	output, err := queryInstallation(brewExecutable, "--prefix", "--installed", "sloctl")
	if err != nil || !isInstalledExecutable(sloctlPath, filepath.Join(strings.TrimSpace(string(output)), "bin")) {
		return updateCommand{}
	}
	return updateCommand{
		display:    "brew upgrade sloctl",
		executable: brewExecutable,
		args:       []string{"upgrade", "sloctl"},
	}
}

func goInstallUpdateCommand(sloctlPath string) updateCommand {
	goExecutable, err := exec.LookPath("go")
	if err != nil || !isGoInstallExecutable(sloctlPath, goExecutable) {
		return updateCommand{}
	}
	return updateCommand{
		display:    "go install github.com/nobl9/sloctl/cmd/sloctl@latest",
		executable: goExecutable,
		args:       []string{"install", "github.com/nobl9/sloctl/cmd/sloctl@latest"},
	}
}

func isGoInstallExecutable(sloctlPath, goExecutable string) bool {
	return isInstalledExecutable(sloctlPath, goBinDir(goExecutable))
}

func isInstalledExecutable(sloctlPath, binDir string) bool {
	if !filepath.IsAbs(binDir) {
		return false
	}
	executableName := "sloctl"
	name := filepath.Base(sloctlPath)
	if runtime.GOOS == "windows" {
		executableName += ".exe"
		name = strings.ToLower(name)
	}
	if name != executableName {
		return false
	}
	file, err := os.Lstat(filepath.Join(binDir, executableName))
	if err != nil || !file.Mode().IsRegular() {
		return false
	}
	// Installers replace a pathname. A hard link in another directory stays outdated.
	return isSameFile(filepath.Dir(sloctlPath), binDir)
}

func goBinDir(goExecutable string) string {
	output, err := queryInstallation(goExecutable, "env", "-json", "GOBIN", "GOPATH")
	if err != nil {
		return ""
	}
	var goEnv struct {
		GOBIN  string `json:"GOBIN"`
		GOPATH string `json:"GOPATH"`
	}
	if err := json.Unmarshal(output, &goEnv); err != nil {
		return ""
	}
	binDir := goEnv.GOBIN
	if binDir == "" {
		goPaths := filepath.SplitList(goEnv.GOPATH)
		if len(goPaths) == 0 || goPaths[0] == "" {
			return ""
		}
		binDir = filepath.Join(goPaths[0], "bin")
	}
	if !filepath.IsAbs(binDir) {
		return ""
	}
	return binDir
}

func queryInstallation(executable string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), installationQueryTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = installationQueryTimeout
	return cmd.Output()
}

func isSameFile(firstPath, secondPath string) bool {
	first, err := os.Stat(firstPath)
	if err != nil {
		return false
	}
	second, err := os.Stat(secondPath)
	if err != nil {
		return false
	}
	return os.SameFile(first, second)
}
