// Copyright (c) the go-webengine/esbuildsandbox authors.
// SPDX-License-Identifier: BSD-3-Clause

package esbuildsandbox

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

func TestResolveDirIsIsolated(t *testing.T) {
	dir := ResolveDir()
	if dir == "" || dir == "/" {
		t.Fatalf("ResolveDir() = %q, want a real non-root path", dir)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("ResolveDir() does not exist as a directory: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read ResolveDir(): %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ResolveDir() has %d entries, want empty", len(entries))
	}
}

func TestNewSandboxDirOrdinaryPath(t *testing.T) {
	got := newSandboxDir(os.MkdirTemp, os.TempDir)
	if got == "" || got == "/" {
		t.Fatalf("newSandboxDir() = %q, want a real non-root path", got)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Fatalf("newSandboxDir() does not exist as a directory: %v", err)
	}
}

func TestNewSandboxDirFallsBackWhenMkdirTempFails(t *testing.T) {
	failingMkdirTemp := func(dir, pattern string) (string, error) {
		return "", errors.New("simulated: temp filesystem exhausted")
	}
	fixedTempDir := func() string { return "/simulated/tmp" }
	if got := newSandboxDir(failingMkdirTemp, fixedTempDir); got != "/simulated/tmp" {
		t.Errorf("newSandboxDir() fallback = %q, want the tempDir() value", got)
	}
}

func TestResolveDirIsMemoized(t *testing.T) {
	a := ResolveDir()
	b := ResolveDir()
	if a != b {
		t.Errorf("ResolveDir() returned %q then %q, want the same path both times", a, b)
	}
}

// TestGlobImportDoesNotWalkRealFilesystem is the regression test: it
// reproduces the actual bug this package fixes by driving a real
// api.Build with a plugin shaped like a typical sandboxing caller's (every
// ordinary import intercepted and served from an in-memory map), and an
// entry script containing a dynamic import whose glob pattern has no fixed
// directory component ahead of its variable segment — the shape that
// esbuild's bundler expands by walking ResolveDir on the real filesystem,
// entirely bypassing the plugin below. With ResolveDir() this must resolve
// to zero matches near-instantly; reverting to a real path such as "/" (or
// simply the process's working directory) reproduces the original bug: the
// build spends seconds recursively walking real directories instead.
func TestGlobImportDoesNotWalkRealFilesystem(t *testing.T) {
	plugin := api.Plugin{
		Name: "in-memory",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				return api.OnResolveResult{Path: args.Path, Namespace: "mem"}, nil
			})
			build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "mem"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				return api.OnLoadResult{}, fmt.Errorf("no such module: %s", args.Path)
			})
		},
	}

	done := make(chan api.BuildResult, 1)
	go func() {
		done <- api.Build(api.BuildOptions{
			Stdin: &api.StdinOptions{
				Contents:   "const id = 'a'; import(`./${id}/index.js`);",
				ResolveDir: ResolveDir(),
				Sourcefile: "entry.js",
				Loader:     api.LoaderJS,
			},
			Bundle:   true,
			Write:    false,
			LogLevel: api.LogLevelSilent,
			Plugins:  []api.Plugin{plugin},
		})
	}()

	select {
	case <-done:
		// A glob import against an empty ResolveDir resolves to zero matches
		// (a warning, not a build error) essentially immediately — completing
		// at all within the bound below is the point; the result's own
		// success/failure is not what this test is guarding.
	case <-time.After(5 * time.Second):
		t.Fatal("api.Build did not complete within 5s — a glob import likely walked real directories, the exact shape of the bug this package fixes")
	}
}
