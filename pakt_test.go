package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCommand(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("sudo dnf install", getCommand("install", []string{"dnf"}))
	assert.Equal("sudo dnf update", getCommand("update", []string{"dnf"}))
	assert.Equal("sudo dnf remove", getCommand("remove", []string{"dnf"}))

	assert.Equal("sudo dnf install && flatpak install", getCommand("install", []string{"dnf", "flatpak"}))
	assert.Equal("sudo dnf update && flatpak update", getCommand("update", []string{"dnf", "flatpak"}))
	assert.Equal("sudo dnf remove && flatpak remove", getCommand("remove", []string{"dnf", "flatpak"}))

	assert.Equal("sudo apt install", getCommand("install", []string{"apt"}))
	assert.Equal("sudo apt update && sudo apt upgrade", getCommand("update", []string{"apt"}))
	assert.Equal("sudo apt remove", getCommand("remove", []string{"apt"}))

	assert.Equal("sudo apt install && flatpak install", getCommand("install", []string{"apt", "flatpak"}))
	assert.Equal("sudo apt update && sudo apt upgrade && flatpak update", getCommand("update", []string{"apt", "flatpak"}))
	assert.Equal("sudo apt remove && flatpak remove", getCommand("remove", []string{"apt", "flatpak"}))
}
