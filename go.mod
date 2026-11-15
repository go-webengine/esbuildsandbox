module github.com/go-webengine/esbuildsandbox

// ⛔ THE PATCH, AND THIS REPOSITORY IS ONE OF THE NINE THE FLEET SWEEPS FOR
// EXPOSURE. This line is the only one of the three Go pins a CONSUMER honours:
// `go install` with GOTOOLCHAIN=local obeys it, while the CI pin merely builds
// our artefacts and the toolchain stamped in a binary describes one executable.
//
// Raised to 1.27.4 on 2026-11-15 for GO-2026-6859 — published 2026-11-11,
// CVE-2026-97054, "HTTP/1 server crash due to MaxBytesReader header race in
// net/http", stdlib, fixed in 1.26.11 and 1.27.4. Read at
// https://vuln.go.dev/ID/GO-2026-6859.json rather than quoted from notes.
//
// Measured, not assumed: `modfloor --floor 1.27.4` read the directive this
// module PUBLISHED at v0.2.0 as `go 1.27.1` — THREE patch releases behind the
// fix, and the furthest behind of the nine. Eight of those nine read BELOW;
// only go-webengine/engine was already at 1.27.4. This is the last of the
// eight.
//
// ⚠ Reachability is govulncheck's question, not this line's: the floor is
// raised because a consumer is entitled to build against the patched library.
//
// ⛔ AND THE CI PIN MOVED WITH IT, in the same change, because it had to: the
// workflow said `go-version: '1.27.3'` — ABOVE this directive and BELOW the
// new one. `setup-go` sets GOTOOLCHAIN=local, so a pin under the directive does
// not fetch a newer toolchain, it REFUSES. Measured the same day on five of six
// go-filesystems repositories, every job red at once. Both steps now read
// `go-version-file: go.mod`, so the next raise needs no edit there.
go 1.27.4

require github.com/evanw/esbuild v0.28.2

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
