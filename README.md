# snspd-go

Go binding for [snspd-lib](https://github.com/altessa-s/snspd-lib), reading
Supernote Atelier `.spd` drawings: canvas metadata, layers, stored PNG tiles,
thumbnail, raw config and viewport-relative bounds.

```sh
go get github.com/altessa-s/snspd-go@latest
```

Package `snspd`; module `github.com/altessa-s/snspd-go`. Requires Go 1.24+, cgo,
and a C23 compiler with typed enums and `nullptr` (clang 19+ or GCC 13+).
No downloads, system SQLite or third-party Go dependencies are needed to build.
For example, select `CC=clang-19` on Linux when the default compiler is older.

Import `github.com/altessa-s/snspd-go` and use `os` and `fmt` for this runnable
example. The file shown is a checked-in synthetic fixture:

```go
data, err := os.ReadFile("testdata/fixtures/valid.spd")
if err != nil {
	panic(err)
}
opts := snspd.DefaultOptions()
opts.Strict = true
opts.RequirePNG128 = true
drawing, err := snspd.Open(data, &opts)
if err != nil {
	panic(err)
}
bounds := drawing.PaintedBounds(nil)
fmt.Printf("%.0f×%.0f canvas, %d layers\n", drawing.Canvas.Width, drawing.Canvas.Height, len(drawing.Layers))
fmt.Printf("stored tile extent: %.0f×%.0f\n", bounds.Width, bounds.Height)
```

`Open(data []byte, opts *Options) (*Drawing, error)` copies every returned
string, tile PNG, thumbnail and raw byte slice into Go memory and frees all C
storage before returning. No `Close` is needed. The input must remain unchanged
during the call; afterward callers may reuse it. Results from independent calls
are independent. Concurrent opens are safe; parsing and conversion are serialized in Go
before entering C, so queued calls hold no C input copy or cgo thread.
Logging runs after C storage is freed and serialization ends. Callers
synchronize mutations of a shared Go model.

`Drawing.Canvas` includes dimensions, PPI and viewport values with explicit
presence flags. Missing/NULL optional strings are nil; stored empty strings
point to `""`. Strings preserve all bytes, including NUL and invalid UTF-8.
`Layers` preserve protobuf order, including empty layers. `Tiles` sort by
`(TX, TY)` or equivalently nonnegative `TID`, in column-major order:
`TID = TX*4096 + TY`, `TX = TID / 4096`, `TY = TID % 4096`.
`TX` is `uint64`; `TY` is `uint32` and ranges from 0 to 4095.
Tile PNG bytes are undecoded. `Layer.Raw` retains the exact
protobuf submessage. Fields 3/4, paper mode and unknown fields have no inferred
visibility, ordering or compositing semantics.

`Config` is in binary key order. `ConfigFind(key)` returns an entry pointer or
nil; it accepts embedded NULs. Keep the slice sorted when modifying it and
remember that a returned pointer refers to its current backing array.
`ConfigEntry.IsNull` and `IsText` preserve SQLite NULL/TEXT/BLOB distinctions.
A stored empty byte value is non-nil; NULL/absent byte values are nil.
`Canvas.Thumbnail` and `TemplateData` remain undecoded/uninterpreted.

`drawing.TileRect(tile)` and `drawing.PaintedBounds(layer)` use stored pixels,
at `X = TX*128 - ViewportX`, `Y = TY*128 - ViewportY`, without applying viewport
scale, a half-page offset or clipping to the canvas. The saved viewport's page
anchoring is unconfirmed: it is not established as the page top-left. A centre
interpretation fits the one available export but remains an inference; see the
[upstream format evidence](https://github.com/altessa-s/snspd-lib/blob/v1.0.0/docs/format.md#tile-grid).
A nil layer includes all layers. Empty extents have `Rect.IsNull` set.
Bounds include transparent tiles; they do not measure nontransparent ink.
Large coordinates can round to double precision; grid spans are calculated
before conversion so a single large tile still has 128-pixel dimensions.

Pass nil options for defaults. Start with `DefaultOptions()` before changing
limits: zero is a literal zero ceiling, including in `Options{}`.
`Strict` requires `fmt_ver="2"` but still accepts unknown fields/keys/tables.
`RequirePNG128` checks only the PNG signature and IHDR dimensions (128×128),
not CRCs, bit depth, color type, decompression or pixels. Default mode retains
and counts mismatches in `InvalidTileCount`. `UnknownFormat` records unknown
or missing format versions.

| Default limit | Value |
|---|---:|
| MaxFileSize | 256 MiB |
| MaxLayers | 256 |
| MaxTilesPerLayer | 262,144 |
| MaxTileBytes / MaxTotalTileBytes | 4 MiB / 256 MiB |
| MaxConfigEntries / MaxConfigBytes | 4,096 / 16 MiB |
| MaxMemoryBytes | 768 MiB |
| MaxWork | 1,073,741,824 |

Limits bound independent resources in the C reader. Memory counts copied input,
SQLite, model and allocation headers; it excludes Go input/output allocations,
host allocator overhead, stack and process-lifetime log bindings. Work counts
retained/examined bytes, allocation requests, protobuf and SQLite VM steps,
lookups and comparisons. It is not a CPU or wall-clock guarantee; lock waits and
callbacks are outside the budget. Raising byte ceilings can require increasing
memory/work ceilings too. SQLite also has fixed record/SQL/schema limits.

Failures return a nil drawing and `*Error`. Use `errors.Is` with
`ErrInvalidArgument`, `ErrOutOfMemory`, `ErrUnsupportedFormat`, `ErrUnexpectedEOF`,
`ErrMalformed` or `ErrLimit`. `errors.As` exposes `Error.Status`, a content-free
`Message` (currently the status phrase for C reader failures), and `Value`
(SQLite result code for engine failures, zero otherwise).
Status and log constants are checked against the C headers at compile time.

`SetLogger(func(LogLevel, string))` routes optional process-wide diagnostics;
nil restores silence. The default is silent. A handler runs in a separate
goroutine synchronously with the open that produced the line; concurrent opens
may call it concurrently. Panics and `runtime.Goexit` drop only that line.
Callbacks may reenter the reader, but must avoid recursive logging and must not
wait for the enclosing open. A line in flight can arrive after silencing.
One C sink is installed for process lifetime; replacing Go handlers does not
accumulate C sink bindings. Messages contain no drawing content, names or paths.

The file must be a complete, self-contained SQLite snapshot. WAL flags are
normalized only in a private input copy and reported in `HadWALHeader`.
Uncheckpointed sidecar transactions cannot be read or detected. SQLite uses a
private metered allocator, a VFS refusing file I/O, readonly deserialization,
defensive mode and no extensions. After fork in a multithreaded process, a child
must exec before using the C library, including logging.

## Vendoring

```sh
scripts/update-clib.sh ../snspd-lib
```

The script refuses dirty checkouts unless `--allow-dirty` is explicit, stages
and validates the copied library, records `VERSION` (`git describe --tags
--always`) and full `COMMIT`, and rolls back `internal/clib`, both cache digest
headers and generated wrappers together on normal failure. It copies the public
headers, C sources, licenses and unmodified SQLite
amalgamation/provenance. It generates one `clib_*.c` wrapper per reader source
and `clib_digest.h`, whose SHA-256 digest forces Go's build cache to notice
nested C changes. The digest test checks every vendored byte and snapshot name.
`Version` is a hand-kept public-header release version, like snnote-go.

Current source: snspd-lib tag **v1.0.0**, header **1.0.0**, commit
`0ac7ba43dcf4b798d9ac5c845dc2eee2b7d45b3b`.
Go module versions are Git tags; no release workflow is provided.

All external reader symbols are renamed with `spdgo_` through
`clib_symbols.h` in C wrappers and cgo preambles. SQLite remains static inside
the parser wrapper, exactly as in the C library; there is no separate SQLite
wrapper in the binding. `clib_sqlite.h` mirrors upstream build options.
`scripts/check-symbols.py` verifies the exact external symbol set. The
`internal/coexist` test package compiles a second ordinary SQLite amalgamation,
keeps a host connection alive and queries it while the binding opens drawings.
This tests both link coexistence and host engine state/allocator independence
without adding a Go dependency. That second wrapper is test support only.

## Tests

```sh
go vet ./...
go test -race -count=1 ./...
GOEXPERIMENT=cgocheck2 go test ./...
golangci-lint run
python3 scripts/check-symbols.py
python3 scripts/test-update-clib.py
go test -run='^$' -fuzz=FuzzOpen -fuzztime=20s .
```

CI follows snnote-go's Linux GCC 13/14, clang 19 and macOS clang matrix, plus
lint. GCC 13 also tests the minimum Go 1.24.x; other entries use stable Go.
With a supported compiler and Go 1.24+ on PATH, the race tests also run offline
with `GOPROXY=off GOTOOLCHAIN=local go test -race ./...`.

Tests cover model ownership, binary config, geometry, extreme tile IDs,
correctly rounded decimals, typed errors, all limits, budget failure cleanup,
hostile schemas/protobuf/B-trees, logging and concurrency. README Go snippets
must match runnable examples. The native Go fuzz corpus and model fixtures come
only from snspd-lib's synthetic generator; no private drawing is included.
See `testdata/README.md` for regeneration.

An opt-in `TestPrivateSampleStructure` reads `SNSPD_SMOKE_FILE` and reports only
counts, dimensions, bounds and byte sizes. It never writes the file or adds it
to the corpus; it is skipped by default.

## License

Copyright 2026 ALTESSA SOLUTIONS INC. See the [MIT license](LICENSE) and
[copyright notice](COPYRIGHT). The bundled SQLite amalgamation is in the
public domain. See [SQLite provenance](internal/clib/sqlite/README.md).

## Trademarks

Supernote is a trademark of Ratta Software. This project is not affiliated
with, endorsed by or sponsored by Ratta Software.
