package internal

import "github.com/spf13/cobra"

func customizeCompletionHelp(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	completion := directSubcommand(root, "completion")
	completion.Short = "Generate shell completion scripts"
	completion.Long = "Generate a completion script for Bash, fish, PowerShell, or Zsh. " +
		"Choose a shell subcommand for loading and installation instructions."

	longDescriptions := map[string]string{
		"bash": "Generate a Bash completion script. Install the " +
			"`bash-completion` package first.\n\n" +
			`**Load completions in the current shell:**

~~~bash
source <(sloctl completion bash)
~~~

**Install completions for future sessions on Linux:**

~~~bash
sloctl completion bash > /etc/bash_completion.d/sloctl
~~~

**With Homebrew on macOS:**

~~~bash
sloctl completion bash > "$(brew --prefix)/etc/bash_completion.d/sloctl"
~~~`,
		"fish": `Generate a fish completion script.

**Load completions in the current shell:**

~~~fish
sloctl completion fish | source
~~~

**Install completions for future sessions:**

~~~fish
sloctl completion fish > ~/.config/fish/completions/sloctl.fish
~~~`,
		"powershell": `Generate a PowerShell completion script.

**Load completions in the current shell:**

~~~powershell
sloctl completion powershell | Out-String | Invoke-Expression
~~~

To persist completions, add that command to your PowerShell profile.`,
		"zsh": "Generate a Zsh completion script.\n\n**Enable completion in `~/.zshrc` if needed:**\n\n" + `~~~zsh
autoload -U compinit
compinit
~~~

**Load completions in the current shell:**

~~~zsh
source <(sloctl completion zsh)
~~~

**Install completions for future sessions on Linux:**

~~~zsh
sloctl completion zsh > "${fpath[1]}/_sloctl"
~~~

**With Homebrew on macOS:**

~~~zsh
sloctl completion zsh > "$(brew --prefix)/share/zsh/site-functions/_sloctl"
~~~`,
	}
	for name, long := range longDescriptions {
		directSubcommand(completion, name).Long = long
	}
}

func directSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	panic("missing Cobra subcommand: " + parent.CommandPath() + " " + name)
}
