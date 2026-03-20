# How MDM Works

MDM manages project documentation in the `.mdm/` directory.

## Structure

```
.mdm/
├── guides/          — instructions for how MDM operates
├── templates/       — templates for generating docs
└── docs/            — generated project documentation
```

## Commands

- `mdm init` — Initialize `.mdm/` in your project (copies default guides and templates)
- `mdm sync-docs` — Use Claude to read the codebase and create/update `.mdm/docs/`
- `mdm update` — Self-update to the latest version
- `mdm version` — Print current version

## Flow

1. Run `mdm init` in your project root
2. Run `mdm sync-docs` to generate documentation
3. Documentation appears in `.mdm/docs/`, following the templates in `.mdm/templates/`
