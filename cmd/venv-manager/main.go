package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jacopobonomi/venv-manager/internal/manager"
	"github.com/spf13/cobra"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorLight  = "\033[98m"
)

var (
	globalFlag bool
	mgr        *manager.Manager
	rootCmd    = &cobra.Command{
		Use:   "venv-manager",
		Short: "A powerful CLI tool for managing Python virtual environments",
		Long:  "venv-manager is a CLI tool for creating, managing, and working with Python virtual environments.",
	}
)

func init() {
	mgr = manager.New("")
	
	rootCmd.PersistentFlags().BoolVar(&globalFlag, "global", false, "Apply command to all environments")
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(removeCmd())
	rootCmd.AddCommand(cloneCmd())
	rootCmd.AddCommand(packagesCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(upgradeCmd())
	rootCmd.AddCommand(cleanCmd())
	rootCmd.AddCommand(activateCmd())
	rootCmd.AddCommand(deactivateCmd())
	rootCmd.AddCommand(completionCmd())
}

func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for venv-manager.
To load completions:

Bash:
  $ source <(venv-manager completion bash)

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ venv-manager completion zsh > "${fpath[1]}/_venv-manager"

Fish:
  $ venv-manager completion fish | source

Powershell:
  PS> venv-manager completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}
	return cmd
}

func createCmd() *cobra.Command {
	var pythonVersion string
	cmd := &cobra.Command{
		Use:   "create <name> [python-version]",
		Short: "Create a new virtual environment",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			if err := mgr.Create(args[0], pythonVersion); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s✨ Created virtual environment '%s'%s\n", colorGreen, args[0], colorReset)
		},
	}
	cmd.Flags().StringVar(&pythonVersion, "python", "", "Python version to use")
	return cmd
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all virtual environments",
		Run: func(cmd *cobra.Command, args []string) {
			venvs, err := mgr.List()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			if len(venvs) == 0 {
				fmt.Printf("%s🌐 No virtual environments found%s\n", colorYellow, colorReset)
				return
			}
			fmt.Printf("%s📂 Available virtual environments:%s\n", colorYellow, colorReset)
			for _, venv := range venvs {
				fmt.Printf("- %s\n", venv)
			}
		},
	}
	return cmd
}

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a virtual environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := mgr.Remove(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s🗑️  Removed virtual environment '%s'%s\n", colorGreen, args[0], colorReset)
		},
	}
	return cmd
}

func cloneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <source> <target>",
		Short: "Clone an existing environment",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := mgr.Clone(args[0], args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s📋 Cloned '%s' to '%s'%s\n", colorGreen, args[0], args[1], colorReset)
		},
	}
	return cmd
}

func packagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packages <name>",
		Short: "List installed packages in an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			packages, err := mgr.ListPackages(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s📦 Packages in '%s':%s\n", colorYellow, args[0], colorReset)
			for _, pkg := range packages {
				fmt.Println(pkg)
			}
		},
	}
	return cmd
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <name> <requirements-file>",
		Short: "Install packages from requirements file",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := mgr.Install(args[0], args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s📦 Installed requirements from '%s' to '%s'%s\n", colorGreen, args[1], args[0], colorReset)
		},
	}
	return cmd
}

func upgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [name]",
		Short: "Upgrade packages in an environment",
		Run: func(cmd *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			name := ""
			if len(args) > 0 && !globalFlag {
				name = args[0]
			}
			if err := mgr.Upgrade(name); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s⬆️  Packages upgraded successfully%s\n", colorGreen, colorReset)
		},
	}
	return cmd
}

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [name]",
		Short: "Clean cache files",
		Run: func(cmd *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			name := ""
			if len(args) > 0 && !globalFlag {
				name = args[0]
			}
			if err := mgr.Clean(name); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%s🧹 Environment cleaned successfully%s\n", colorGreen, colorReset)
		},
	}
	return cmd
}

func activateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Activate a virtual environment",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			venvPath := filepath.Join(mgr.GetBaseDir(), args[0])
			if _, err := os.Stat(venvPath); os.IsNotExist(err) {
				fmt.Printf("%s❌ Venv '%s' does not exist%s\n", colorRed, args[0], colorReset)
				os.Exit(1)
			}
			fmt.Printf("source %s/bin/activate", venvPath)
		},
	}
	return cmd
}

func deactivateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate",
		Short: "Deactivate the current virtual environment",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("deactivate")
		},
	}
	return cmd
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s%v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}