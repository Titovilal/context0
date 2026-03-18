# Releasing MDM

## Publish a new version

```bash
# 1. Make sure you're on main with everything committed
git checkout main
git pull

# 2. Tag the version
git tag v0.1.0

# 3. Push the tag — this triggers the GitHub Actions build
git push origin v0.1.0
```

The workflow compiles binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and creates a GitHub Release automatically.

## Commit naming

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add background task queue
fix: agent not resuming after rewind
refactor: rename ctm to mdm
docs: update README installation
chore: add release workflow
```

Prefixes: `feat`, `fix`, `refactor`, `docs`, `chore`, `test`, `ci`.

## Version numbering

Use semver: `vMAJOR.MINOR.PATCH`

- **patch** (v0.1.1): bug fixes
- **minor** (v0.2.0): new features, new commands, new connectors
- **major** (v1.0.0): breaking changes (CLI flags, registry format, etc.)

## Verify the release

After pushing the tag, check:
1. https://github.com/Titovilal/middleman/actions — workflow should be green
2. https://github.com/Titovilal/middleman/releases — release should have 4 binaries

## Test the install script

```bash
curl -sL https://raw.githubusercontent.com/Titovilal/middleman/main/install.sh | sh
mdm --help
```

## Hotfix a release

```bash
# Fix the bug, commit
git add .
git commit -m "fix: whatever broke"

# Tag the patch
git tag v0.1.1
git push origin main v0.1.1
```

## Delete a bad release

```bash
# Delete remote tag and re-push
git push --delete origin v0.1.0
git tag -d v0.1.0

# Then delete the release from GitHub UI or:
gh release delete v0.1.0 --yes
```
