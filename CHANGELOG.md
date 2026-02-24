# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-02-23

### Added
- `Query` — execute JSONPath against raw JSON bytes
- `QueryContext` — context-aware query execution
- `QueryValue` / `QueryValueContext` — query against pre-parsed Go values
- `First` — return first match or nil
- `Values` — return just values from results
- `Paths` — return just paths from results
- `Exists` — check if any match exists
- `MustQuery` — panic-on-error variant for tests
- `Compile` / `MustCompile` — pre-compile paths for reuse
- `CompiledPath.Query` / `CompiledPath.QueryContext` — execute pre-compiled path
- `Result` type with `Path` and `Value` fields and `MarshalJSON` support
- Full JSONPath syntax: root, child, wildcard, index, negative index, slice, step slice, recursive descent, filter, union
- Filter expressions: comparisons (`<`, `>`, `<=`, `>=`, `==`, `!=`), existence, regex (`=~`), logical AND/OR
- Structured `Error` type with `ErrorCode`
- `IsPathError`, `IsJSONError`, `IsFilterError`, `IsNotFound`, `IsCancelled` helpers
- `WithMaxDepth` option for recursive descent
- `WithAllowMissingKeys` option for strict mode
- Deterministic output (sorted object keys)
- GitHub Actions CI with Go 1.21/1.22/1.23 matrix
- Full test coverage including benchmarks
