# Pakt

Track and sync packages from system package manager or other package manager (flatpak).

## Usage
Installing a package:
```
pakt install vim
```

Installing a package using other package manager (flatpak):
```
pakt install -f chromium
```

Sync from package.json file:
```
pakt sync
```

For a the complete list of commands, run `pakt --help`.

Packages installed using pakt is tracked in `$HOME/.config/pakt`.

## Installation

1. Clone the repository.
2. Make sure Go is installed, then run `go install`.