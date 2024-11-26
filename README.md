# fsplit

`fsplit` is a tool to split Go files into single function files to maximize the GitHub Copilot experience.

## Installation

```sh
go install github.com/nakario/fsplit/cmd/fsplit@latest
```

## Usage

```sh
fsplit <package-path>
```

Replace `<package-path>` with the path to the Go package you want to split.

## Features

- Extracts functions from the package and creates single function files.
- Removes functions from the original files.
- Excludes test files and generated files.
- Skips files with one or fewer functions.

## License

This project is licensed under the MIT License.
