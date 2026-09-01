# go-webengine / esbuildsandbox

[![CI](https://github.com/go-webengine/esbuildsandbox/actions/workflows/ci.yml/badge.svg)](https://github.com/go-webengine/esbuildsandbox/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-webengine/esbuildsandbox.svg)](https://pkg.go.dev/github.com/go-webengine/esbuildsandbox)
[![License: BSD-3-Clause](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](LICENSE)

A safe `ResolveDir` for [`evanw/esbuild`](https://github.com/evanw/esbuild)'s
`pkg/api`, closing a sandbox leak that is easy to introduce by accident and
easy to miss in review.

```go
res := api.Build(api.BuildOptions{
    Stdin: &api.StdinOptions{
        Contents:   entrySource,
        ResolveDir: esbuildsandbox.ResolveDir(), // not "/", not "."
        Sourcefile: "entry.js",
        Loader:     api.LoaderJS,
    },
    Plugins: []api.Plugin{yourPlugin},
})
```

Set it on every `ResolveDir` esbuild's `pkg/api` asks for —
`Stdin.ResolveDir` and every `OnLoadResult.ResolveDir` a plugin returns.

## The problem

A caller that drives esbuild programmatically — bundling untrusted or remote
source through a `Plugin`, the way a headless browser engine bundles a web
page's own `<script type="module">` graph — typically installs an
`OnResolve`/`OnLoad` plugin that intercepts every ordinary import and serves
it from wherever the real source actually lives (a network fetch, a virtual
filesystem, an in-memory map). That plugin looks complete: every import in
the program's own test suite goes through it.

But esbuild's bundler special-cases one import shape the plugin API never
sees: a **glob dynamic import** — `` import(`./${lang}/index.js`) `` or
`` require(`./chunks/${id}.js`) ``, ordinary output from a bundler's own
locale/route code-splitting, nothing exotic or malicious. The bundler expands
a glob import by walking `ResolveDir` directly on the **real filesystem** via
its own internal resolver, entirely bypassing `OnResolve` and `OnLoad`.
Whatever `ResolveDir` is set to — `"/"` is a common default, or simply
whatever directory the host process happens to be running in — is what gets
walked, following symlinks, looking for "matches". A plugin that correctly
sandboxes every ordinary import still leaves this one path wide open, and the
caller does not find out until something builds a page containing exactly
this (very common) import shape.

This was found and fixed in
[`go-webengine/engine`](https://github.com/go-webengine/engine): a
production page (`developer.mozilla.org`) containing a glob-shaped dynamic
import made that engine's renderer spend ~9 seconds almost entirely in
`readdir`/symlink syscalls walking the host's real root filesystem, for a
**single page render** — a genuine sandbox leak, not merely a slowdown.

## The fix

`ResolveDir()` returns a directory that stays empty for the life of the
process (created once, lazily, via `os.MkdirTemp`). Point every `ResolveDir`
esbuild sees at it instead of `"/"` or a working directory: a glob import
then resolves to zero matches immediately — the same outcome a browser's own
sandboxed module loader would produce — rather than touching the real
filesystem at all.

## License

BSD-3-Clause © the go-webengine/esbuildsandbox authors.
