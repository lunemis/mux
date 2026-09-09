# Contributing to tmux-peeker

Thanks for your interest in contributing! Here's how to get started.

## Development setup

```bash
git clone https://github.com/aemonge/tmux-peeker.git
cd tmux-peeker
make build
```

### Prerequisites

- Go 1.24.2+
- tmux

### Running tests

```bash
make test
```

### Building

```bash
make build    # builds ./tmux-peeker binary
make install  # installs to /usr/local/bin
```

## Making changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit with a descriptive message
6. Push to your fork and open a Pull Request

## Commit messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add session sorting options
fix: correct preview rendering on narrow terminals
docs: update keybinding table
```

## Code style

- Run `go vet ./...` and `golangci-lint run` before submitting
- Follow standard Go conventions
- Keep functions focused and small

### Theme colors

All UI colors live in [`theme/*.json`](theme/). Add new semantic color roles to each palette, expose them through `theme.Colors`, and reference those roles from the UI instead of hard-coding colors in Go. Built-in themes are embedded into the binary at build time.

## Reporting issues

- Search existing issues before opening a new one
- Include your OS, Go version, and tmux version
- Describe the expected vs actual behavior

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
