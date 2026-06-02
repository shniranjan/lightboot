# Tool Knowledge Base

## `read_file`
- **Status**: ✅ Works perfectly
- **Usage**: Read any file in the project. Always works.
- **Notes**: Use before editing any file. Always verify content first.

## `single_find_and_replace`
- **Status**: ⚠️ Partially works
- **Issues**:
  - `old_string` must match EXACTLY what's in the file, character-for-character
  - Multi-line strings with `\n` often fail to match
  - When searching for Go code with tabs (`\t`), the tool sometimes doesn't match
  - The `\"` escaping inside old_string/new_string can cause confusion
- **Best practices**:
  - Use for single-line replacements only
  - Avoid escaped quotes in old_string; use exact file content
  - Verify file content with `read_file` first, copy text exactly
  - For multi-line changes, prefer `run_terminal_command` with a Python script
- **Example that worked**: `single_find_and_replace` with `"d.conn.SetReadDeadline(nil)"` → `"// read without deadline..."`
- **Example that worked**: Fixing comment block for SetReadDeadline

## `edit_existing_file`
- **Status**: ❌ Not reliable
- **Issues**: Failed with "Cannot read properties of undefined" and other errors
- **Best practice**: Avoid this tool; use `single_find_and_replace` or terminal commands instead

## `create_new_file`
- **Status**: ⚠️ Partially works
- **Issues**: 
  - Sometimes wraps content in extra double quotes (`"..."`)
  - Can create files but content may be corrupted with escaped quotes
- **Best practice**: Use for simple files; verify content after creation with `read_file`
- **Files created**: `tmp/fix_main.py`, `tmp/fix_dhcp.py`
- **Issue with fix_dhcp.py**: Got wrapped in `"..."` making the Python file start with `"#!/usr...` (syntax error)

## `run_terminal_command`
"- **Status**: ✅ Works
- **🚨 CRITICAL**: NEVER wrap command in outer double quotes. That makes bash treat the entire string as a filename. Use `cd /path && cmd`, NOT `\"cd /path && cmd\"`"
- **Issues**:
  - Commands wrapped in `"..."` outer quotes cause `No such file or directory` errors
  - Heredocs (`<< 'EOF'`) fail with `No such file or directory`
  - Complex Python inline scripts with nested quotes fail
  - Simple commands without special characters work fine
- **Best practices**:
  - Use WITHOUT outer double quotes around the whole command
  - Example that works: `cd /path && go build ./... 2>&1 | head -30`
  - Example that works: `grep -n "pattern" file.go`
  - Example that FAILS: `"cd /path && command"` (outer quotes break it)
  - For Python scripts: write to a .py file first (via create_new_file or heredoc to /tmp), then execute
  - Use `cat > /tmp/script.py << 'ENDPYTHON'` pattern (without outer quotes)
  - Use `\n` for newlines in Python one-liners (not actual newlines in command string)

## `file_glob_search`
- **Status**: ✅ Works
- **Example**: `file_glob_search` with `fix_main*` found `tmp/fix_main.py`

## `grep_search`
- **Status**: ✅ Works (used successfully)
- **Best practice**: Use for finding code patterns across the project

## `view_diff`
- **Status**: Not tested in this session
- **Notes**: Should work for viewing git changes

## `read_currently_open_file`
- **Status**: Not tested
- **Notes**: Use when user is looking at a file but hasn't specified the path

## `fetch_url_content`
- **Status**: Not tested

## `ls`
- **Status**: ✅ Works
- **Best practice**: Use for listing directory contents

## `read_skill`
- **Status**: Not tested

---

## Summary of Known Issues
1. **Multi-line string matching in single_find_and_replace**: Consistently fails. Use Python scripts instead.
2. **Outer quotes in run_terminal_command**: Commands should NOT be wrapped in outer double quotes.
3. **create_new_file quote wrapping**: Content sometimes gets extra quotes. Always verify after creation.
4. **Shell heredocs**: Unreliable. Prefer writing Python scripts to /tmp and executing them.
5. **Escaped quotes in old_string/new_string**: The tool treats `\"` literally sometimes, causing mismatches.

## Stage 5 Learnings (Vue.js + Vite)
- **Vite build error "make/index.html"**: When running `make web`, the Makefile used `cd web && npm install && npx vite build` which should work. But running `npx vite build` from the project root without `cd web` first causes Vite to look for `index.html` in the wrong directory. Always run Vite from the `web/` directory.
- **Vite build succeeded**: `cd web && npx vite build` works perfectly. 34 modules transformed, output to `web/dist/`.
- **TFTP shutdown panic**: The `pin/tftp` library can panic in `Shutdown()` if `ListenAndServe` hasn't fully initialized. Fix: add a readiness channel in `Start()` that signals after the goroutine has started, and wrap `Shutdown()` with a `recover()` call.
- **SSE (EventSource) cannot send Authorization headers**: The live logs stream at `/api/logs/stream` must be a public endpoint. Auth is handled via the token check on login, not on the SSE connection.
- **Go 1.22+ routing patterns**: `mux.HandleFunc("GET /api/isos", ...)` supports method routing. More specific patterns (like `GET /api/logs/stream`) take precedence over prefix patterns (like `mux.Handle("/api/", authedAPI)`). This is how we keep SSE public while the rest of `/api/` is authenticated.
- **SPA handler needs both http.FileSystem and fs.FS**: The fallback handler serves static files via `http.FileServer` but needs `fs.ReadFile` for the index.html fallback. Store both interfaces.


## Stage 5 Learnings (Vue.js + Vite)
- **Vite build error "make/index.html"**: When running `make web`, the Makefile used `cd web && npm install && npx vite build` which should work. But running `npx vite build` from the project root without `cd web` first causes Vite to look for `index.html` in the wrong directory. Always run Vite from the `web/` directory.
- **Vite build succeeded**: `cd web && npx vite build` works perfectly. 34 modules transformed, output to `web/dist/`.
- **TFTP shutdown panic**: The `pin/tftp` library can panic in `Shutdown()` if `ListenAndServe` hasn't fully initialized. Fix: add a readiness channel in `Start()` that signals after the goroutine has started, and wrap `Shutdown()` with a `recover()` call.
- **SSE (EventSource) cannot send Authorization headers**: The live logs stream at `/api/logs/stream` must be a public endpoint. Auth is handled via the token check on login, not on the SSE connection.
- **Go 1.22+ routing patterns**: `mux.HandleFunc("GET /api/isos", ...)` supports method routing. More specific patterns (like `GET /api/logs/stream`) take precedence over prefix patterns (like `mux.Handle("/api/", authedAPI)`). This is how we keep SSE public while the rest of `/api/` is authenticated.
- **SPA handler needs both http.FileSystem and fs.FS**: The fallback handler serves static files via `http.FileServer` but needs `fs.ReadFile` for the index.html fallback. Store both interfaces.

## Stage 6 Learnings (MkDocs + Go Embed + License)
- **MkDocs site_dir restriction**: The `site_dir` cannot be nested inside `docs_dir` or mkdocs aborts. Solution: set `site_dir: site` at the project root.
- **MkDocs missing icon**: `material/boot` doesn't exist in older MkDocs Material. Use `material/lightbulb` instead.
- **Dead doc links**: A link to a missing `CODE_OF_CONDUCT.md` causes a mkdocs warning but doesn't abort. Fix by linking to GitHub repo.
- **Go embed parent restriction**: `//go:embed` only works with subdirectories, not `..`. Solution: place the embed file at project root to embed `site/`.
- **Root package import**: A Go file at project root with `package lightboot` can be imported from internal packages as `root "github.com/.../lightboot"`.
- **Docs route precedence**: `/docs/` registered BEFORE the SPA `/*` fallback on the mux, so documentation pages take priority.
- **Python fix pattern**: Write .py files first, then execute with `python3 /path/script.py`. Avoid complex inline shell quoting.
- **License**: Created GPLv2 LICENSE with prominent disclaimer of warranty at the top. Updated index.md footer and contributing.md.
- **Clean binary test**: `make build` produces working binary. `/docs/` returns 200. Full HTML documentation served from embedded FS.
