# go-opsiscript-lsp

A Go-based language server for `opsi-script` / Opsiscript.

This project provides a standalone LSP server over stdio for editor integration. It currently focuses on parsing, navigation, diagnostics, hover, and completion for real-world Opsiscript package sources.

## Related projects

- [nvim-opsiscript](https://github.com/ChristianBoehm/nvim-opsiscript)
- [opsi-script manual](https://docs.opsi.org/opsi-docs-en/4.3/opsi-script-manual/opsi-script-manual.html)
- Coming soon - [Opsiscript syntax for VS Code](https://github.com/ChristianBoehm/vscode-opsiscript-syntax)

## Current features

- full-document text sync over stdio
- diagnostics for:
  - malformed section headers
  - duplicate sections
  - duplicate variables
  - unknown section calls
  - malformed `if` / `elseif` / `else` / `endif` block structure
  - lightweight conditional expression errors
- hover for:
  - variables
  - sections
  - builtin commands
  - builtin functions
  - builtin constants
  - user-defined functions
- completion for:
  - builtin commands/functions/constants
  - section names
  - variables
  - conditional snippets like `if`, `ifelse`, `elseif`, `else`, `endif`
- document symbols for sections, functions, and variables
- go-to-definition and find-references for sections, variables, and user-defined functions
- include/import resolution across files for:
  - `include_insert`
  - `include_append`
  - `importlib`

## Install

### With `go install`

```bash
go install github.com/ChristianBoehm/go-opsiscript-lsp/cmd/go-opsiscript-lsp@latest
```

This installs the `go-opsiscript-lsp` binary into your Go bin directory.

### From source

```bash
git clone https://github.com/ChristianBoehm/go-opsiscript-lsp.git
cd go-opsiscript-lsp
go build ./cmd/go-opsiscript-lsp
```

## Build

```bash
go build ./cmd/go-opsiscript-lsp
```

This produces the `go-opsiscript-lsp` binary in the repository root when built locally with the current setup.

## Run

The server is intended to be started by an editor or LSP client over stdio.

Example:

```bash
./go-opsiscript-lsp
```

## Neovim example

**File to edit:** `~/.config/nvim/init.lua`

**Option to add** alongside your existing `nvim-lspconfig` setup:

```lua
local capabilities = require("cmp_nvim_lsp").default_capabilities()
capabilities.textDocument.completion.completionItem.snippetSupport = true

vim.lsp.config("opsiscript_lsp", {
  cmd = { "/path/to/go-opsiscript-lsp/go-opsiscript-lsp" },
  filetypes = { "opsiscript" },
  root_markers = { ".git" },
  capabilities = capabilities,
  flags = { debounce_text_changes = 200 },
})

vim.lsp.enable("opsiscript_lsp")
```

You also need an Opsiscript filetype plugin, for example [`nvim-opsiscript`](https://github.com/ChristianBoehm/nvim-opsiscript).

## Project layout

- `cmd/go-opsiscript-lsp/` - binary entrypoint
- `internal/lexer/` - line/token-oriented lexical analysis
- `internal/parser/` - Opsiscript parsing and reference extraction
- `internal/analysis/` - diagnostics and semantic checks
- `internal/symbols/` - builtin metadata and symbol indexing
- `internal/lsp/` - JSON-RPC and LSP server implementation
- `internal/files/` - URI/path and include resolution helpers

## Scope

This is currently a language server, not an Opsiscript interpreter. It aims to improve editor support first and expand language coverage incrementally using the official docs plus real-world package scripts.

## Module

```text
github.com/ChristianBoehm/go-opsiscript-lsp
```

## License

MIT
