# Contributing

Thank you for your interest in contributing to go-jsonpath!

## Development

This project requires no external dependencies. Standard Go tooling is sufficient.
```bash
go test ./...
go vet ./...
gofmt -l .
```

## Guidelines

- All public APIs must have GoDoc comments with examples where appropriate
- All new features must include table-driven tests
- Filter and path behaviour should match the [JSONPath RFC 9535](https://www.rfc-editor.org/rfc/rfc9535) where possible
- Maintain zero external dependencies
- Maintain Go 1.21+ compatibility

## Pull Requests

1. Fork the repository
2. Create a feature branch
3. Write tests for new behaviour
4. Run `go test ./...` and `go vet ./...`
5. Submit a PR with a clear description

## Reporting Issues

Please open a GitHub issue with:
- Your Go version
- The JSONPath expression that caused the problem
- A minimal JSON document to reproduce
- Expected vs actual output
```

---
