# SVG Reflect

Small Go web service that renders configured request data as SVG.

## Run

```powershell
go run .
```

The service reads `config.yaml` from the current directory by default. Set
`SVG_REFLECT_CONFIG` to use another config path, or `SVG_REFLECT_LISTEN` to
override the configured listen address for a local run.

## Endpoints

```text
GET /svg/{profile}.svg
```

Examples:

```text
http://localhost:8080/svg/default.svg?user=alice&trace=123
http://localhost:8080/svg/debug.svg?trace=abc
```

Profile names must contain only letters, digits, `_`, and `-`.

Long text, header values, and query values wrap automatically inside the SVG
width, and the SVG height grows to fit the wrapped lines.
