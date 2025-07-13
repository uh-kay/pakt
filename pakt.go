package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
)

type Data struct {
	PackageManagers map[string][]string `json:"package_managers"`
}

type PackageManager struct {
	name      string
	commands  map[string]string
	needsSudo bool
}

var packageManagers = map[string]PackageManager{
	"dnf": {
		name:      "dnf",
		needsSudo: true,
		commands: map[string]string{
			"install": "install",
			"remove":  "remove",
			"update":  "update",
		},
	},
	"apt": {
		name:      "apt",
		needsSudo: true,
		commands: map[string]string{
			"install": "install",
			"remove":  "remove",
			"update":  "update && sudo apt upgrade",
		},
	},
	"pacman": {
		name:      "pacman",
		needsSudo: true,
		commands: map[string]string{
			"install": "-S",
			"remove":  "-R",
			"update":  "-Syu",
		},
	},
	"flatpak": {
		name:      "flatpak",
		needsSudo: false,
		commands: map[string]string{
			"install": "install",
			"remove":  "remove",
			"update":  "update",
		},
	},
	"nix": {
		name:      "nix",
		needsSudo: false,
		commands: map[string]string{
			"install": "profile add nixpkgs#",
			"remove":  "profile remove",
			"update":  "profile upgrade --all",
		},
	},
}

func main() {
	app := &cli.Command{
		Name:  "pakt",
		Usage: "track and sync packages",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "flatpak",
				Aliases: []string{"f"},
				Usage:   "set package manager to flatpak",
			},
			&cli.BoolFlag{
				Name:    "nix",
				Aliases: []string{"n"},
				Usage:   "set package manager to nix",
			},
			&cli.BoolFlag{
				Name:    "update-all",
				Aliases: []string{"a"},
				Usage:   "update all packages",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "install",
				Aliases: []string{"i"},
				Usage:   "install a package",
				Action: func(c context.Context, app *cli.Command) error {
					return runCommand(app)
				},
			},
			{
				Name:    "remove",
				Usage:   "remove a package",
				Aliases: []string{"r"},
				Action: func(c context.Context, app *cli.Command) error {
					return runCommand(app)
				},
			},
			{
				Name:    "update",
				Usage:   "update a package",
				Aliases: []string{"u"},
				Action: func(c context.Context, app *cli.Command) error {
					return runCommand(app)
				},
			},
			{
				Name:  "sync",
				Usage: "sync packages",
				Action: func(c context.Context, app *cli.Command) error {
					return sync()
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func sync() error {
	data, err := loadPackageData()
	if err != nil {
		return err
	}

	for packageManagerName, packages := range data.PackageManagers {
		if len(packages) == 0 {
			continue
		}

		command := getCommand("install", []string{packageManagerName})
		if command == "" {
			fmt.Printf("Unknown package manager: %s\n", packageManagerName)
			continue
		}

		commandParts := append(strings.Fields(command), packages...)

		if err := executeCommand(commandParts); err != nil {
			fmt.Printf("Error installing packages with %s: %v\n", packageManagerName, err)
		}
	}

	return nil
}

func loadPackageData() (*Data, error) {
	filename, err := getPackageFilePath()
	if err != nil {
		return nil, fmt.Errorf("getting package file path: %w", err)
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Cannot sync: package.json not found")
			return &Data{PackageManagers: make(map[string][]string)}, nil
		}
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var data Data
	if err := json.Unmarshal(fileData, &data); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &data, nil
}

func savePackageData(data *Data) error {
	filename, err := getPackageFilePath()
	if err != nil {
		return fmt.Errorf("getting package file path: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func addPackageToFile(packageManagerName, packageName string) error {
	data, err := loadPackageData()
	if err != nil {
		return err
	}

	if data.PackageManagers == nil {
		data.PackageManagers = make(map[string][]string)
	}

	// Check if package already exists
	if slices.Contains(data.PackageManagers[packageManagerName], packageName) {
		return nil
	}

	data.PackageManagers[packageManagerName] = append(
		data.PackageManagers[packageManagerName],
		packageName,
	)

	return savePackageData(data)
}

func removePackageFromFile(packageManagerName, packageName string) error {
	data, err := loadPackageData()
	if err != nil {
		return err
	}

	if data.PackageManagers == nil {
		return nil
	}

	packages := data.PackageManagers[packageManagerName]
	if index := slices.Index(packages, packageName); index != -1 {
		data.PackageManagers[packageManagerName] = slices.Delete(packages, index, index+1)
		return savePackageData(data)
	}

	return nil
}

func getPackageManager() []string {
	cmd := exec.Command("sh", "-c", `grep '^ID=' /etc/os-release | cut -d'=' -f2 | tr -d '\n'`)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error detecting distro: %v", err)
		return []string{}
	}

	distroName := strings.TrimSpace(string(output))

	switch distroName {
	case "fedora":
		return []string{"dnf"}
	case "ubuntu", "linuxmint":
		return []string{"apt"}
	case "arch":
		return []string{"pacman"}
	default:
		log.Printf("Unknown distro: %s", distroName)
		return []string{}
	}
}

func getCommand(action string, packageManagerNames []string) string {
	if len(packageManagerNames) == 0 {
		return ""
	}

	var commands []string

	for _, pmName := range packageManagerNames {
		pm, exists := packageManagers[pmName]
		if !exists {
			continue
		}

		actionCmd, exists := pm.commands[action]
		if !exists {
			continue
		}

		var cmd string
		if pm.needsSudo {
			cmd = "sudo " + pm.name + " " + actionCmd
		} else {
			cmd = pm.name + " " + actionCmd
		}

		commands = append(commands, cmd)
	}

	return strings.Join(commands, " && ")
}

func executeCommand(commandParts []string) error {
	if len(commandParts) == 0 {
		return fmt.Errorf("no command to execute")
	}

	cmd := exec.Command(commandParts[0], commandParts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runCommand(app *cli.Command) error {
	packageName := app.Args().Get(0)
	if packageName == "" && app.Name != "update" {
		return fmt.Errorf("package name is required")
	}

	var packageManagerNames []string

	switch {
	case app.Bool("flatpak"):
		packageManagerNames = []string{"flatpak"}
	case app.Bool("nix"):
		packageManagerNames = []string{"nix"}
	case app.Bool("update-all"):
		packageManagerNames = append(getPackageManager(), "flatpak")
	default:
		packageManagerNames = getPackageManager()
	}

	if len(packageManagerNames) == 0 {
		return fmt.Errorf("no package manager detected")
	}

	command := getCommand(app.Name, packageManagerNames)
	if command == "" {
		return fmt.Errorf("unsupported action or package manager")
	}

	var commandParts []string

	if packageName != "" {
		if packageManagerNames[0] == "nix" && app.Name == "install" {
			nixCmd := strings.Fields(command)
			nixCmd[len(nixCmd)-1] = nixCmd[len(nixCmd)-1] + packageName
			commandParts = nixCmd
		} else {
			commandParts = append(strings.Fields(command), packageName)
		}
	} else {
		commandParts = strings.Fields(command)
	}

	if err := executeCommand(commandParts); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	switch app.Name {
	case "install":
		if packageName != "" {
			if err := addPackageToFile(packageManagerNames[0], packageName); err != nil {
				fmt.Printf("Warning: failed to track package: %v\n", err)
			}
		}
	case "remove":
		if packageName != "" {
			if err := removePackageFromFile(packageManagerNames[0], packageName); err != nil {
				fmt.Printf("Warning: failed to untrack package: %v\n", err)
			}
		}
	}

	return nil
}

func getPackageFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "pakt")
	return filepath.Join(configDir, "package.json"), nil
}
