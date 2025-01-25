package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jacopobonomi/venv-manager/internal/manager"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorLight  = "\033[98m"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	global := false
	args := os.Args[1:]
	if args[0] == "--global" {
		global = true
		args = args[1:]
	}

	mgr := manager.New("")
	mgr.SetGlobal(global)
	switch args[0] {
	case "activate":
		if len(args) < 2 {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}

		venvPath := filepath.Join(mgr.GetBaseDir(), args[1])
		if _, err := os.Stat(venvPath); os.IsNotExist(err) {
			fmt.Printf("%s❌ Venv '%s' does not exist%s\n", colorRed, args[1], colorReset)
			os.Exit(1)
		}

		fmt.Printf("Run: source %s/bin/activate", venvPath)

	case "deactivate":
		fmt.Println("Run: deactivate")

	case "packages":
		if len(args) < 2 {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		packages, err := mgr.ListPackages(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s📦 Packages in '%s':%s\n", colorYellow, args[1], colorReset)
		for _, pkg := range packages {
			fmt.Println(pkg)
		}

	case "install":
		if len(args) < 3 {
			fmt.Printf("%s❌ Error: Missing venv name or requirements file%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Install(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s📦 Installed requirements from '%s' to '%s'%s\n", colorGreen, args[2], args[1], colorReset)

	case "clone":
		if len(args) < 3 {
			fmt.Printf("%s❌ Error: Missing source or target venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Clone(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s📋 Cloned '%s' to '%s'%s\n", colorGreen, args[1], args[2], colorReset)

	case "upgrade":
		if len(args) < 2 && !global {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		name := ""
		if !global {
			name = args[1]
		}
		if err := mgr.Upgrade(name); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s⬆️  Packages upgraded successfully%s\n", colorGreen, colorReset)

	case "clean":
		if len(args) < 2 && !global {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		name := ""
		if !global {
			name = args[1]
		}
		if err := mgr.Clean(name); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s🧹 Environment cleaned successfully%s\n", colorGreen, colorReset)

	case "create":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		pythonVersion := ""
		if len(os.Args) > 3 {
			pythonVersion = os.Args[3]
		}
		if err := mgr.Create(os.Args[2], pythonVersion); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s✨ Created virtual environment '%s'%s\n", colorGreen, args[1], colorReset)

	case "list":
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

	case "remove":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Missing venv name%s\n", colorRed, colorReset)
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Remove(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s🗑️  Removed virtual environment '%s'%s\n", colorGreen, args[1], colorReset)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}
func printUsage() {
	fmt.Printf("\n%s📚 Usage:%s\n", colorCyan, colorReset)
	fmt.Printf("  %s[--global] venv-manager <command> [arguments]%s\n\n", colorLight, colorReset)

	fmt.Printf("%s🛠️  Commands:%s\n", colorCyan, colorReset)
	commands := []struct {
		cmd, args, desc, emoji string
	}{
		{"create", "<name> [python-version]", "Create new virtual environment", "🆕"},
		{"list", "", "List all environments", "📋"},
		{"remove", "<name>", "Remove environment", "🗑️"},
		{"clone", "<source> <target>", "Clone environment", "📋"},
		{"packages", "<name>", "List installed packages", "📦"},
		{"install", "<name> <requirements>", "Install from requirements", "⬇️"},
		{"upgrade", "<name>", "Upgrade all packages", "⬆️"},
		{"clean", "<name>", "Clean cache files", "🧹"},
		{"activate", "<name>", "Activate environment", "▶️"},
		{"deactivate", "", "Deactivate environment", "⏹️"},
	}

	for _, cmd := range commands {
		fmt.Printf("  %s%-10s %-20s %s %s%s\n",
			colorLight, cmd.cmd, cmd.args, cmd.emoji, cmd.desc, colorReset)
	}

	fmt.Printf("\n%s🚩 Flags:%s\n", colorCyan, colorReset)
	fmt.Printf("  %s--global%20s🌐 Apply to all environments%s\n",
		colorLight, "", colorReset)
}
