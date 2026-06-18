# skillbelt

A package manager for agent skills. Installs skills by cloning git repositories and symlinking them into your harness skills directory.

By default skillbelt targets [Antigravity](https://antigravity.google). Support for other agent harnesses is coming soon — if your harness uses the same skill directory format it should already work by pointing `SKILLBELT_SKILLS_DIR` at the right path.

## Installation

**Download a release binary** (Linux and macOS, amd64 and arm64):

Download the archive for your platform from the [releases page](https://github.com/carlosafonso/skillbelt/releases), extract it, and move the binary to somewhere on your `PATH`.

macOS (Apple Silicon):
```
curl -L https://github.com/carlosafonso/skillbelt/releases/latest/download/skillbelt_darwin_arm64.tar.gz | tar -xz
mv skillbelt /usr/local/bin/
```

macOS (Intel):
```
curl -L https://github.com/carlosafonso/skillbelt/releases/latest/download/skillbelt_darwin_amd64.tar.gz | tar -xz
mv skillbelt /usr/local/bin/
```

Linux (amd64):
```
curl -L https://github.com/carlosafonso/skillbelt/releases/latest/download/skillbelt_linux_amd64.tar.gz | tar -xz
mv skillbelt /usr/local/bin/
```

Linux (arm64):
```
curl -L https://github.com/carlosafonso/skillbelt/releases/latest/download/skillbelt_linux_arm64.tar.gz | tar -xz
mv skillbelt /usr/local/bin/
```

**Install with Go:**

```
go install github.com/carlosafonso/skillbelt/cmd/skillbelt@latest
```

## Usage

### Install a skill

```
skillbelt install <url>
```

Accepts a full repo URL or a GitHub subdirectory URL:

```
skillbelt install github.com/someone/my-skill
skillbelt install https://github.com/someone/monorepo/tree/main/skills/my-skill
```

For subdirectory URLs, only that directory is fetched via sparse checkout.

### List installed skills

```
skillbelt list
```

### Update skills

```
skillbelt update              # update all
skillbelt update <name>       # update one
```

### Remove a skill

```
skillbelt remove <name>           # removes the symlink, keeps the local clone
skillbelt remove <name> --purge   # removes the symlink and the local clone
```

## Development

### Running tests

```
go test ./...
```

Or via Make:

```
make test
```

### Git hooks

A pre-commit hook that runs the test suite is provided in `scripts/pre-commit`. Install it once after cloning:

```
make install-hooks
```

### CI

Tests run automatically on every push and pull request targeting `main` via GitHub Actions.

## Configuration

| Environment variable    | Default                      | Description                        |
|-------------------------|------------------------------|------------------------------------|
| `SKILLBELT_SKILLS_DIR`  | `~/.gemini/config/skills`    | Where skill symlinks are created   |
| `SKILLBELT_HOME`        | `~/.skillbelt`               | Where repos and the lock file live |
