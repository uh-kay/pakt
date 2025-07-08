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
				Name:    "update-all",
				Aliases: []string{"a"},
				Usage:   "update all packages",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "install",
				Usage: "install a package",
				Action: func(c context.Context, app *cli.Command) error {
					return runCommand(app)
				},
			},
			{
				Name:  "remove",
				Usage: "remove a package",
				Action: func(c context.Context, app *cli.Command) error {
					return runCommand(app)
				},
			},
			{
				Name:  "update",
				Usage: "update a package",
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
	filename, err := getPackageFilePath()
	if err != nil {
		fmt.Println("Error getting package file path:", err)
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Cannot sync: package.json not found")

			return nil
		}

		fmt.Println("Error reading file:", err)

		return nil
	}

	var data Data
	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
	}

	for packageManager, packages := range data.PackageManagers {
		var commandParts []string

		command := getCommand("install", []string{packageManager})

		commandParts = strings.Fields(command)

		for _, packageName := range packages {
			commandParts = append(commandParts, packageName)
		}

		cmd := exec.Command(commandParts[0], commandParts[1:]...)

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()
	}

	return nil
}

func writeToFile(packageManager []string, packageName string) {
	var data Data

	filename, err := getPackageFilePath()
	if err != nil {
		fmt.Println("Error getting package file path:", err)
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			data = Data{
				PackageManagers: make(map[string][]string),
			}

			data.PackageManagers[packageManager[0]] = append(data.PackageManagers[packageManager[0]], packageName)

			jsonData, err := json.MarshalIndent(data, "", "    ")
			if err != nil {
				fmt.Println("Error marshaling JSON:", err)
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				panic(err)
			}

			configDir := filepath.Join(homeDir, ".config", "pakt")
			filename := filepath.Join(configDir, "package.json")

			err = os.MkdirAll(configDir, 0755)
			if err != nil {
				panic(err)
			}

			err = os.WriteFile(filename, jsonData, 0644)
			if err != nil {
				fmt.Println("Error writing file:", err)
			}

			return
		}

		fmt.Println("Error reading file:", err)
	}

	err = json.Unmarshal(fileData, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
	}

	if slices.Contains(data.PackageManagers[packageManager[0]], packageName) {
		return
	}

	data.PackageManagers[packageManager[0]] = append(data.PackageManagers[packageManager[0]], packageName)

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
	}
}

func getPackageManager() []string {
	var packageManager []string

	cmd := exec.Command("sh", "-c", `grep '^ID=' /etc/os-release | cut -d'=' -f2 | tr -d '\n'`)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	distroName := string(output[:])

	switch distroName {
	case "fedora":
		packageManager = []string{"dnf"}
	case "ubuntu", "linuxmint":
		packageManager = []string{"apt"}
	}

	return packageManager
}

func getCommand(action string, packageManager []string) string {
	var command string

	if len(packageManager) > 1 {
		var commandPart string
		switch packageManager[0] {
		case "dnf":
			commandPart = "sudo " + packageManager[0] + " " + action
		case "apt":
			if action == "update" {
				commandPart = "sudo " + packageManager[0] + " " + action + " && " + "sudo " + packageManager[0] + " " + "upgrade"
			} else {
				commandPart = "sudo " + packageManager[0] + " " + action
			}
		}
		command = commandPart + " && " + packageManager[1] + " " + action
		return command
	}

	switch packageManager[0] {
	case "dnf":
		command = "sudo " + packageManager[0] + " " + action

	case "flatpak":
		command = "flatpak " + " " + action

	case "apt":
		switch action {
		case "install":
			command = "sudo " + packageManager[0] + " " + action
		case "remove":
			command = "sudo " + packageManager[0] + " " + action
		case "update":
			command = "sudo " + packageManager[0] + " " + action + " && " + "sudo " + packageManager[0] + " " + "upgrade"
		}
	}

	return command
}

func runCommand(app *cli.Command) error {
	var packageManager []string

	packageName := app.Args().Get(0)

	if app.Bool("flatpak") {
		packageManager = []string{"flatpak"}
	} else if app.Bool("update-all") {
		packageManager = append(getPackageManager(), "flatpak")
	} else {
		packageManager = getPackageManager()
	}

	command := getCommand(app.Name, packageManager)

	cmd := exec.Command("sh", "-c", command+" "+packageName)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return err
	}

	switch app.Name {
	case "install":
		writeToFile(packageManager, packageName)
	case "remove":
	case "update":
	}

	return nil
}

func getPackageFilePath() (string, error) {
	configDir := makeDir()

	return filepath.Join(configDir, "package.json"), nil
}

func makeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configDir := filepath.Join(homeDir, ".config", "pakt")

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	return configDir
}
