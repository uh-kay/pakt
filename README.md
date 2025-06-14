# Pakt

Track and sync packages from system package manager or other package manager (flatpak).

## Usage
Installing a package:
```
pakt install vim
```

Installing a package using other package manager:
```
pakt install -m flatpak chromium
```

For a the complete list of commands, run `pakt --help`.

## Installation

1. Clone the repository.
2. Make sure Go is installed, then run `go install`.