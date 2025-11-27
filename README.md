<div align="center">
  <h1> TreeGo
 </h1>
</div>

<p align="center">
  <img src="https://img.shields.io/github/stars/marcuwynu23/treego.svg" alt="Stars Badge"/>
  <img src="https://img.shields.io/github/forks/marcuwynu23/treego.svg" alt="Forks Badge"/>
  <img src="https://img.shields.io/github/issues/marcuwynu23/treego.svg" alt="Issues Badge"/>
  <img src="https://img.shields.io/github/license/marcuwynu23/treego.svg" alt="License Badge"/>
</p>

TreeGo is a fast, concurrent, and safe directory tree printer and file search tool written in Go. It allows you to print the directory structure, search for files, and filter results using regex patterns or directories-only mode.

---

## Features

- Print a visual directory tree.
- Search for files and directories by name.
- Filter files and directories using regex.
- Option to display directories only.
- Safe concurrent traversal with automatic error handling.
- Exits immediately if an error or deadlock is detected.

---

## Installation

1. Ensure you have [Go](https://golang.org/dl/) installed (1.20+ recommended).
2. Clone the repository:

```bash
git clone https://github.com/marcuwynu23/treego.git
cd treego
```

3. Build the binary:

```bash
go build -o treego main.go
```

4. Run the tool:

```bash
./treego <path> [--search <query>] [--regex <pattern>] [--dirs-only]
```

---

## Usage

```text
treego <path> [--search <query>] [--regex <pattern>] [--dirs-only] [--version]
```

### Flags

- `--search`, `-s` : Search string. Prints full path of matching files.
- `--regex`, `-r` : Regex filter to match file or directory names.
- `--dirs-only`, `-d` : Show only directories.
- `--version` : Show TreeGo version.

### Examples

Print the tree of a folder:

```bash
treego /path/to/project
```

Search for files containing `main`:

```bash
treego /path/to/project --search main
```

Show directories only:

```bash
treego /path/to/project --dirs-only
```

Use regex to filter names:

```bash
treego /path/to/project --regex "\.go$"
```

---

## Safety Features

- Uses a global abort channel to exit immediately on errors.
- Prevents deadlocks when traversing large directories.
- Concurrent processing of directories for faster traversal.

---
