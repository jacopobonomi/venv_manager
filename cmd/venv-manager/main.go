package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jacopobonomi/venv-manager/internal/manager"
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
			fmt.Println("Error: Missing venv name")
			printUsage()
			os.Exit(1)
		}

		venvPath := filepath.Join(mgr.GetBaseDir(), args[1])
		if _, err := os.Stat(venvPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Venv '%s' does not exist\n", args[1])
			os.Exit(1)
		}

		fmt.Printf("source %s/bin/activate", venvPath)

	case "install":
		if len(args) < 3 {
			fmt.Println("Error: Missing venv name or requirements file")
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Install(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Installed requirements from '%s' to '%s'\n", args[2], args[1])
	case "clone":
		if len(args) < 3 {
			fmt.Println("Error: Missing source or target venv name")
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Clone(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Cloned '%s' to '%s'\n", args[1], args[2])

	case "upgrade":
		if len(args) < 2 && !global {
			fmt.Println("Error: Missing venv name")
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
		fmt.Println("Packages upgraded successfully")

	case "clean":
		if len(args) < 2 && !global {
			fmt.Println("Error: Missing venv name")
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
		fmt.Println("Environment cleaned successfully")
	case "create":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing venv name")
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
		fmt.Printf("Created virtual environment '%s'\n", os.Args[2])

	case "list":
		venvs, err := mgr.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if len(venvs) == 0 {
			fmt.Println("No virtual environments found")
			return
		}
		fmt.Println("Available virtual environments:")
		for _, venv := range venvs {
			fmt.Printf("- %s\n", venv)
		}

	case "remove":
		if len(os.Args) < 3 {
			fmt.Println("Error: Missing venv name")
			printUsage()
			os.Exit(1)
		}
		if err := mgr.Remove(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed virtual environment '%s'\n", os.Args[2])

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  [--global] venv-manager <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  create <name> [python-version]  - Create a new virtual environment")
	fmt.Println("  activate <name>                 - Print command to activate environment")
	fmt.Println("  install <name>                 - Install packages from requirements file")
	fmt.Println("  list                           - List all virtual environments")
	fmt.Println("  remove <name>                  - Remove a virtual environment")
	fmt.Println("  clone <source> <target>        - Clone a virtual environment")
	fmt.Println("  upgrade <name>                 - Upgrade all packages in environment")
	fmt.Println("  clean <name>                   - Clean cache and temporary files")
	fmt.Println("\nFlags:")
	fmt.Println("  --global                       - Apply command to all environments")
}
