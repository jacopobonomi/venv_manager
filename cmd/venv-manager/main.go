package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jacopobonomi/venv-manager/internal/config"
	"github.com/jacopobonomi/venv-manager/internal/manager"
	"github.com/jacopobonomi/venv-manager/internal/tui"
	"github.com/jacopobonomi/venv-manager/internal/utils"
	"github.com/spf13/cobra"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

var (
	globalFlag bool
	jsonFlag   bool
	mgr        *manager.Manager
	cfg        *config.Config

	rootCmd = &cobra.Command{
		Use:   "venv-manager",
		Short: "A powerful CLI tool for managing Python virtual environments",
		Long:  "venv-manager creates, manages, and works with Python virtual environments.",
	}
)

func init() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%swarning: failed to load config: %v%s\n", colorYellow, err, colorReset)
		cfg = &config.Config{}
	}
	mgr = manager.NewWithOptions(manager.Options{
		BaseDir:       cfg.BaseDir,
		DefaultPython: cfg.DefaultPython,
		UseUv:         cfg.UseUv,
	})

	rootCmd.PersistentFlags().BoolVar(&globalFlag, "global", false, "Apply command to all environments")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	rootCmd.AddCommand(
		createCmd(), listCmd(), removeCmd(), renameCmd(), cloneCmd(),
		packagesCmd(), installCmd(), upgradeCmd(), cleanCmd(),
		activateCmd(), deactivateCmd(), sizeCmd(),
		runCmd(), doctorCmd(), pruneCmd(), exportCmd(), importCmd(),
		configCmd(), tuiCmd(), completionCmd(),
	)
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "%s%v%s\n", colorRed, err, colorReset)
	os.Exit(1)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		die(err)
	}
}

func createCmd() *cobra.Command {
	var pythonVersion string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new virtual environment",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Create(args[0], pythonVersion); err != nil {
				die(err)
			}
			fmt.Printf("%s✨ Created virtual environment '%s'%s\n", colorGreen, args[0], colorReset)
		},
	}
	cmd.Flags().StringVar(&pythonVersion, "python", "", "Python version to use (e.g. 3.12)")
	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all virtual environments",
		Run: func(_ *cobra.Command, _ []string) {
			venvs, err := mgr.List()
			if err != nil {
				die(err)
			}
			if jsonFlag {
				printJSON(venvs)
				return
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
}

func removeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a virtual environment",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Remove(args[0]); err != nil {
				die(err)
			}
			fmt.Printf("%s🗑️  Removed virtual environment '%s'%s\n", colorGreen, args[0], colorReset)
		},
	}
}

func renameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a virtual environment",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Rename(args[0], args[1]); err != nil {
				die(err)
			}
			fmt.Printf("%s✏️  Renamed '%s' to '%s'%s\n", colorGreen, args[0], args[1], colorReset)
		},
	}
}

func cloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <source> <target>",
		Short: "Clone an existing environment",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Clone(args[0], args[1]); err != nil {
				die(err)
			}
			fmt.Printf("%s📋 Cloned '%s' to '%s'%s\n", colorGreen, args[0], args[1], colorReset)
		},
	}
}

func packagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "packages <name>",
		Short: "List installed packages in an environment",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			packages, err := mgr.ListPackages(args[0])
			if err != nil {
				die(err)
			}
			if jsonFlag {
				printJSON(packages)
				return
			}
			fmt.Printf("%s📦 Packages in '%s':%s\n", colorYellow, args[0], colorReset)
			for _, pkg := range packages {
				fmt.Println(pkg)
			}
		},
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <name> <requirements-file>",
		Short: "Install packages from requirements file",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Install(args[0], args[1]); err != nil {
				die(err)
			}
			fmt.Printf("%s📦 Installed requirements from '%s' to '%s'%s\n", colorGreen, args[1], args[0], colorReset)
		},
	}
}

func upgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [name]",
		Short: "Upgrade packages in an environment",
		Run: func(_ *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			name := ""
			if len(args) > 0 && !globalFlag {
				name = args[0]
			}
			if err := mgr.Upgrade(name); err != nil {
				die(err)
			}
			fmt.Printf("%s⬆️  Packages upgraded successfully%s\n", colorGreen, colorReset)
		},
	}
}

func cleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clean [name]",
		Short: "Clean cache files",
		Run: func(_ *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			name := ""
			if len(args) > 0 && !globalFlag {
				name = args[0]
			}
			if err := mgr.Clean(name); err != nil {
				die(err)
			}
			fmt.Printf("%s🧹 Environment cleaned successfully%s\n", colorGreen, colorReset)
		},
	}
}

func activateCmd() *cobra.Command {
	var shell string
	cmd := &cobra.Command{
		Use:   "activate <name>",
		Short: "Print the shell command to activate a venv (use with `eval $(...)` or `source <(...)`)",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			out, err := mgr.GetActivationCommand(args[0], shell)
			if err != nil {
				die(err)
			}
			fmt.Print(out)
		},
	}
	cmd.Flags().StringVar(&shell, "shell", os.Getenv("SHELL"), "Shell name (bash, zsh, fish, pwsh, cmd)")
	return cmd
}

func deactivateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate",
		Short: "Print the deactivate command",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("deactivate")
		},
	}
}

func sizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "size [name]",
		Short: "Check the size of a virtual environment",
		Run: func(_ *cobra.Command, args []string) {
			mgr.SetGlobal(globalFlag)
			name := ""
			if len(args) > 0 && !globalFlag {
				name = args[0]
			}
			sizes, err := mgr.GetSize(name)
			if err != nil {
				die(err)
			}
			if jsonFlag {
				printJSON(sizes)
				return
			}
			if name != "" {
				fmt.Printf("%s📊 Size of '%s':%s\n", colorYellow, name, colorReset)
			} else {
				fmt.Printf("%s📊 Sizes of all virtual environments:%s\n", colorYellow, colorReset)
			}
			for venvName, size := range sizes {
				fmt.Printf("- %s: %s\n", venvName, utils.FormatSize(size))
			}
		},
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <name> -- <command> [args...]",
		Short: "Run a command inside a venv without activating it",
		Args:  cobra.MinimumNArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			if err := mgr.Run(args[0], args[1:]); err != nil {
				die(err)
			}
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the venv-manager environment",
		Run: func(_ *cobra.Command, _ []string) {
			r := mgr.Doctor()
			if jsonFlag {
				printJSON(r)
				return
			}
			fmt.Printf("%s🩺 venv-manager doctor%s\n", colorYellow, colorReset)
			fmt.Printf("  Base dir       : %s (exists=%v)\n", r.BaseDir, r.BaseDirExists)
			fmt.Printf("  Config file    : %s\n", config.Path())
			fmt.Printf("  uv available   : %v\n", r.UvAvailable)
			fmt.Printf("  Python found   : %v\n", r.PythonVersions)
			fmt.Printf("  Venvs          : %d\n", r.VenvCount)
			if len(r.Broken) > 0 {
				fmt.Printf("  %sBroken venvs   : %v%s\n", colorRed, r.Broken, colorReset)
			}
		},
	}
}

func pruneCmd() *cobra.Command {
	var days int
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove venvs unused for a given number of days",
		Run: func(_ *cobra.Command, _ []string) {
			if days == 0 {
				days = cfg.PruneAfterDays
			}
			stale, err := mgr.FindStale(days)
			if err != nil {
				die(err)
			}
			if jsonFlag {
				printJSON(stale)
				return
			}
			if len(stale) == 0 {
				fmt.Printf("%s✨ No stale venvs (older than %d days)%s\n", colorGreen, days, colorReset)
				return
			}
			fmt.Printf("%s🗑️  Stale venvs (older than %d days):%s\n", colorYellow, days, colorReset)
			for _, s := range stale {
				fmt.Printf("- %s (last modified %s)\n", s.Name, s.ModTime.Format("2006-01-02"))
			}
			if dryRun {
				return
			}
			for _, s := range stale {
				if err := mgr.Remove(s.Name); err != nil {
					fmt.Fprintf(os.Stderr, "%sfailed to remove %s: %v%s\n", colorRed, s.Name, err, colorReset)
					continue
				}
				fmt.Printf("%s✅ Removed %s%s\n", colorGreen, s.Name, colorReset)
			}
		},
	}
	cmd.Flags().IntVar(&days, "days", 0, "Age threshold in days (defaults to config prune_after_days)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only report; do not remove")
	return cmd
}

func exportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <name>",
		Short: "Export a venv manifest as JSON to stdout",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			mf, err := mgr.Export(args[0])
			if err != nil {
				die(err)
			}
			printJSON(mf)
		},
	}
}

func importCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <manifest.json>",
		Short: "Recreate a venv from a manifest file",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			data, err := os.ReadFile(args[0])
			if err != nil {
				die(err)
			}
			var mf manager.Manifest
			if err := json.Unmarshal(data, &mf); err != nil {
				die(err)
			}
			if err := mgr.Import(&mf); err != nil {
				die(err)
			}
			fmt.Printf("%s📥 Imported venv '%s'%s\n", colorGreen, mf.Name, colorReset)
		},
	}
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print the current config",
		Run: func(_ *cobra.Command, _ []string) {
			printJSON(cfg)
			fmt.Fprintf(os.Stderr, "config file: %s\n", config.Path())
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Run: func(_ *cobra.Command, _ []string) { fmt.Println(config.Path()) },
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Write a default config file if none exists",
		Run: func(_ *cobra.Command, _ []string) {
			if _, err := os.Stat(config.Path()); err == nil {
				die(fmt.Errorf("config already exists at %s", config.Path()))
			}
			if err := cfg.Save(); err != nil {
				die(err)
			}
			fmt.Printf("%s✅ Wrote %s%s\n", colorGreen, config.Path(), colorReset)
		},
	})
	return cmd
}

func tuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI to browse and manage virtual environments",
		Run: func(_ *cobra.Command, _ []string) {
			if err := tui.Run(mgr); err != nil {
				die(err)
			}
		},
	}
}

func completionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion scripts",
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		die(err)
	}
}
