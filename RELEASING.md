# Releasing MDM

## Publish a new version

```bash
# 1. Make sure you're on main with everything committed
git checkout main
git pull

# 2. Tag the version (YY.M.D.N)
git tag 26.3.18.1

# 3. Push the tag — this triggers the GitHub Actions build
git push origin 26.3.18.1
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

Format: `YY.M.D.N` — year, month, day, release number of the day.

Examples:
- `26.3.18.1` — first release on March 18, 2026
- `26.3.18.2` — second release on March 18, 2026
- `26.4.1.1` — first release on April 1, 2026

No `v` prefix.

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

# Tag with incremented N
git tag 26.3.18.2
git push origin main 26.3.18.2
```

## Delete a bad release

```bash
# Delete remote tag and re-push
git push --delete origin 26.3.18.1
git tag -d 26.3.18.1

# Then delete the release from GitHub UI or:
gh release delete 26.3.18.1 --yes
```
