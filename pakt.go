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
		command := getCommand("install", packageManager)

		for _, packageName := range packages {
			commandParts := strings.Fields(command)

			commandParts = append(commandParts, packageName)

			cmd := exec.Command(commandParts[0], commandParts[1:]...)

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			_ = cmd.Run()
		}
	}

	return nil
}

func writeToFile(packageManager string, packageName string) {
	data := Data{
		PackageManagers: make(map[string][]string),
	}

	filename, err := getPackageFilePath()
	if err != nil {
		fmt.Println("Error getting package file path:", err)
	}

	fileData, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			data.PackageManagers[packageManager] = append(data.PackageManagers[packageManager], packageName)

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

	if slices.Contains(data.PackageManagers[packageManager], packageName) {
		return
	}

	data.PackageManagers[packageManager] = append(data.PackageManagers[packageManager], packageName)

	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Println("Error:", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
	}
}

func getPackageManager() string {
	var packageManager string

	cmd := exec.Command("sh", "-c", `cat /etc/os-release | grep -oP 'NAME="\K[^"]*'`)
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	distroName := strings.Split(string(output), " ")[0]

	switch distroName {
	case "Fedora":
		packageManager = "dnf"
	}

	return packageManager
}

func getCommand(action string, packageManager string) string {
	var command string

	switch packageManager {
	case "dnf":
		command = "sudo " + packageManager + " " + action

	case "flatpak", "f":
		command = "flatpak " + " " + action
	}

	return command
}

func runCommand(app *cli.Command) error {
	var packageManager string

	packageName := app.Args().Get(0)

	if app.Bool("flatpak") {
		packageManager = "flatpak"
	} else {
		packageManager = getPackageManager()
	}

	command := getCommand(app.Name, packageManager)
	commandParts := strings.Fields(command)

	if !(app.Name == "update" && packageName == "") {
		commandParts = append(commandParts, packageName)
	}

	cmd := exec.Command(commandParts[0], commandParts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	_ = cmd.Run()

	writeToFile(packageManager, packageName)

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
