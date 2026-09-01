// Copyright (c) the go-webengine/esbuildsandbox authors.
// SPDX-License-Identifier: BSD-3-Clause

// Package esbuildsandbox provides a safe ResolveDir for github.com/evanw/
// esbuild's pkg/api, closing a sandbox leak that is easy to introduce by
// accident and easy to miss in review.
//
// # The problem
//
// A caller that drives esbuild programmatically — bundling untrusted or
// remote source through a Plugin, the way a headless browser engine bundles
// a web page's own <script type="module"> graph — typically installs an
// OnResolve/OnLoad plugin that intercepts every ordinary import and serves
// it from wherever the real source actually lives (a network fetch, a
// virtual filesystem, an in-memory map). That plugin looks complete: every
// import in the program's own test suite goes through it.
//
// But esbuild's bundler special-cases one import shape that the plugin API
// does not see at all: a *glob* dynamic import — `import(`./${lang}/index.js`)`
// or `require(`./chunks/${id}.js`), ordinary output from a bundler's own
// locale/route code-splitting, nothing exotic or malicious. The bundler
// expands a glob import by walking ResolveDir directly on the REAL
// filesystem via its own internal resolver, entirely bypassing OnResolve
// and OnLoad. Whatever ResolveDir is set to — "/" is a common default, or
// simply whatever directory the host process happens to be running in — is
// what gets walked, following symlinks, looking for "matches". A plugin
// that correctly sandboxes every ordinary import still leaves this one path
// wide open, and the caller does not find out until something builds a page
// containing exactly this (very common) import shape.
//
// This was found and fixed in github.com/go-webengine/engine: a production
// page (developer.mozilla.org) containing a glob-shaped dynamic import made
// that engine's renderer spend ~9 seconds almost entirely in readdir/
// symlink syscalls walking the host's real root filesystem, for a single
// page render — a genuine sandbox leak, not merely a slowdown.
//
// # The fix
//
// Point every ResolveDir esbuild sees — Stdin.ResolveDir and every
// OnLoadResult.ResolveDir a plugin returns — at [ResolveDir]'s directory
// instead. It is created once (lazily, on first call) via os.MkdirTemp and
// never written to by this package, so it stays empty for the life of the
// process: any glob import resolves to zero matches immediately, the same
// outcome a browser's own sandboxed module loader would produce, rather
// than touching the real filesystem at all.
package esbuildsandbox

import (
	"os"
	"sync"
)

var (
	once sync.Once
	dir  string
)

// ResolveDir returns a directory that stays empty for the life of the
// process, safe to use as every ResolveDir esbuild's pkg/api asks for
// (api.StdinOptions.ResolveDir, api.OnLoadResult.ResolveDir, and any other
// field of that shape). It is created once, lazily, and the same path is
// returned on every subsequent call.
//
// Nothing is ever written into the returned directory by this package, so a
// glob-shaped dynamic import esbuild resolves against it always finds zero
// matches — a real filesystem location, but a permanently empty one, rather
// than the host's actual root or working directory.
//
// If the directory cannot be created (an exhausted or unwritable temp
// filesystem), ResolveDir falls back to os.TempDir(): not guaranteed empty,
// but still a far smaller, more contained tree than a real filesystem root,
// and a strictly better default than reintroducing the sandbox leak this
// package exists to close.
func ResolveDir() string {
	once.Do(func() {
		dir = newSandboxDir(os.MkdirTemp, os.TempDir)
	})
	return dir
}

// newSandboxDir is ResolveDir's logic with its two os calls as parameters,
// so both the ordinary and the fallback path are exercised directly by a
// test rather than only the ordinary one (mkdirTemp failing needs an
// exhausted or unwritable temp filesystem to occur naturally).
func newSandboxDir(mkdirTemp func(dir, pattern string) (string, error), tempDir func() string) string {
	d, err := mkdirTemp("", "esbuildsandbox-")
	if err != nil {
		return tempDir()
	}
	return d
}
