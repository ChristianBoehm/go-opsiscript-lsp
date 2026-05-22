# go-opsiscript-lsp

A Go-based language server for `opsi-script` / Opsiscript.

Provides a standalone LSP server over stdio for editor integration. Focuses on parsing, navigation, diagnostics, hover, and completion for real-world Opsiscript package sources.

## Related projects

- [nvim-opsiscript](https://github.com/ChristianBoehm/nvim-opsiscript)
- [opsi-script manual](https://docs.opsi.org/opsi-docs-en/4.3/opsi-script-manual/opsi-script-manual.html)
- Coming soon: [Opsiscript syntax for VS Code](https://github.com/ChristianBoehm/vscode-opsiscript-syntax)

## Features

- Full-document text sync over stdio
- Diagnostics for:
  - malformed section headers
  - duplicate sections and variables
  - unknown section calls
  - malformed `if` / `elseif` / `else` / `endif` block structure
  - lightweight conditional expression errors
- Hover for variables, sections, builtin commands, builtin functions, builtin constants, and user-defined functions
- Completion for builtin commands/functions/constants, section names, variables, and conditional snippets (`if`, `ifelse`, `elseif`, `else`, `endif`)
- Document symbols for sections, functions, and variables
- Go-to-definition and find-references for sections, variables, and user-defined functions
- Include/import resolution across files for `include_insert`, `include_append`, and `importlib`

## Requirements

- Go 1.26 or later
- Communicates over stdio — intended to be launched by an editor or LSP client, not run as a daemon

## Install

### With `go install`

```
go install github.com/ChristianBoehm/go-opsiscript-lsp/cmd/go-opsiscript-lsp@latest
```

This installs the binary into your Go bin directory. The full path is:

```
$(go env GOPATH)/bin/go-opsiscript-lsp
```

If `$(go env GOPATH)/bin` is on your `$PATH`, you can reference the binary simply as `go-opsiscript-lsp` in your editor config.

### From source

```
git clone https://github.com/ChristianBoehm/go-opsiscript-lsp.git
cd go-opsiscript-lsp
go build -o go-opsiscript-lsp ./cmd/go-opsiscript-lsp
```

This produces the `go-opsiscript-lsp` binary in the current directory.

## Editor integration

### Neovim

Requires **Neovim 0.11 or later** and a filetype plugin such as
[nvim-opsiscript](https://github.com/ChristianBoehm/nvim-opsiscript).

**Using nvim-opsiscript's helper** (recommended):

```lua
local capabilities = require("cmp_nvim_lsp").default_capabilities()
capabilities.textDocument.completion.completionItem.snippetSupport = true

vim.lsp.config("opsiscript_lsp", require("opsiscript").lsp_config(
  "/path/to/go-opsiscript-lsp",
  { capabilities = capabilities }
))

vim.lsp.enable("opsiscript_lsp")
```

**Manual setup:**

```lua
local capabilities = require("cmp_nvim_lsp").default_capabilities()
capabilities.textDocument.completion.completionItem.snippetSupport = true

vim.lsp.config("opsiscript_lsp", {
  cmd = { "/path/to/go-opsiscript-lsp" },
  filetypes = { "opsiscript", "opsiinc" },
  root_markers = { ".git", "OPSI", "CLIENT_DATA" },
  capabilities = capabilities,
  flags = { debounce_text_changes = 200 },
})

vim.lsp.enable("opsiscript_lsp")
```

Replace `/path/to/go-opsiscript-lsp` with the actual binary path, e.g.
`/home/user/go/bin/go-opsiscript-lsp` or the path from your local build.

## Project layout

```
cmd/go-opsiscript-lsp/   binary entrypoint
internal/lexer/          line/token-oriented lexical analysis
internal/parser/         Opsiscript parsing and reference extraction
internal/analysis/       diagnostics and semantic checks
internal/symbols/        builtin metadata and symbol indexing
internal/lsp/            JSON-RPC and LSP server implementation
internal/files/          URI/path and include resolution helpers
```

## Module

```
github.com/ChristianBoehm/go-opsiscript-lsp
```

## Scope

This is a language server, not an Opsiscript interpreter. It aims to improve editor support first and expand language coverage incrementally using the official docs and real-world package scripts.

## License

MIT
