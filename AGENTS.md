# AGENTS

This file provides instructions and context for agentic coding agents working on the `facefeed` repository.

## Project Overview

`facefeed` is a Go-based CLI tool for publishing content to Facebook Pages. It supports text, links, and multiple images (including automatic SVG-to-PNG conversion). It features an Anthropic-styled UI for a professional terminal experience.

## Build, Lint, and Test Commands

### Build and Run

- **Build the binary:** `go build -o facefeed main.go`
- **Run without building:** `go run main.go [args]`
- **Install dependencies:** `go mod tidy`

### Linting and Formatting

- **Format code:** `go fmt ./...`
- **Run static analysis:** `go vet ./...`
- **Lint (if golangci-lint is installed):** `golangci-lint run`

### Testing

- **Run all tests:** `go test ./...`
- **Run a specific test:** `go test -v -run TestName ./path/to/pkg`
- **Run tests with coverage:** `go test -cover ./...`

## Code Style Guidelines

### General

- Follow standard Go idioms and conventions (see [Effective Go](https://golang.org/doc/effective_go)).
- Use `gofmt` for consistent formatting.

### Imports

- Group imports into two blocks:
  1. Standard library packages.
  2. Third-party packages and internal modules.
- Use meaningful aliases if a package name is ambiguous or doesn't match its purpose (e.g., `theme "facefeed/internal"`).

### Naming Conventions

- **Files:** Use lowercase with underscores if needed (e.g., `ui_helper.go`), though Go usually prefers short, single-word names.
- **Packages:** Short, lowercase, single-word names.
- **Functions/Types:** Use `PascalCase` for exported identifiers and `camelCase` for unexported ones.
- **Variables:** Use `camelCase`. Prefer short names for local variables with limited scope (e.g., `i` for index, `r` for reader).
- **Receivers:** Use 1-3 letter abbreviations (e.g., `func (il *ImageList) ...`).

### Error Handling

- Errors should be handled explicitly. Do not ignore errors using `_` unless there is a very strong justification.
- Wrap errors with context to aid debugging: `fmt.Errorf("failed to process image %s: %w", path, err)`.
- In `main.go`, use the `theme` package to report errors to the user before exiting.

### Types and Structs

- Group related fields together.
- Use struct tags (e.g., `json:"id"`) for serialization.
- For custom flags, implement the `flag.Value` interface (see `ImageList` in `main.go`).

### CLI UI & Output

- All user-facing output should use the `theme` package located in `internal/` (aliased as `theme`).
- **Available UI components:**
  - `theme.PrintHeader()`: Displays the ASCII art logo.
  - `theme.PrintSection(title)`: Starts a new section in the output.
  - `theme.Info(label, value)`: Prints a labeled information line.
  - `theme.Success(msg)`: Prints a success message with a checkmark.
  - `theme.Warning(msg)`: Prints a warning message.
  - `theme.Error(msg)`: Prints an error message with an "X" mark.
  - `theme.NewProgressBar(size, desc)`: Returns a progress bar for long-running tasks.

## Environment Configuration

- The project uses `.env` files for configuration via `github.com/joho/godotenv`.
- Required variables: `FB_PAGE_ID`, `FB_ACCESS_TOKEN`.
- Optional variables: `FB_MESSAGE`, `FB_IMAGES`.

---

<skills_system priority="1">

## Available Skills

<!-- SKILLS_TABLE_START -->
<usage>
When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively. Skills provide specialized capabilities and domain knowledge.

How to use skills:

- Invoke: Bash("openskills read <skill-name>")
- The skill content will load with detailed instructions on how to complete the task
- Base directory provided in output for resolving bundled resources (references/, scripts/, assets/)

Usage notes:

- Only use skills listed in <available_skills> below
- Do not invoke a skill that is already loaded in your context
- Each skill invocation is stateless
  </usage>

<available_skills>

<skill>
<name>algorithmic-art</name>
<description>Creating algorithmic art using p5.js with seeded randomness and interactive parameter exploration. Use this when users request creating art using code, generative art, algorithmic art, flow fields, or particle systems. Create original algorithmic art rather than copying existing artists' work to avoid copyright violations.</description>
<location>project</location>
</skill>

<skill>
<name>brand-guidelines</name>
<description>Applies Anthropic's official brand colors and typography to any sort of artifact that may benefit from having Anthropic's look-and-feel. Use it when brand colors or style guidelines, visual formatting, or company design standards apply.</description>
<location>project</location>
</skill>

<skill>
<name>skill-creator</name>
<description>Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.</description>
<location>project</location>
</skill>

</available_skills>

<!-- SKILLS_TABLE_END -->

</skills_system>
