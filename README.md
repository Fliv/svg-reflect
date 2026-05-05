# SVG Reflect Worker

Cloudflare Worker that renders configured request data as SVG.

## Endpoints

```text
GET /svg/{profile}.svg
```

Examples:

```text
https://your-worker.example/svg/default.svg?user=alice&trace=123
https://your-worker.example/svg/debug.svg?trace=abc
```

Profile names must contain only letters, digits, `_`, and `-`.

Long text, header values, and query values wrap automatically inside the SVG
width, and the SVG height grows to fit the wrapped lines.

## Configure

Edit `src/config.js`. Each profile under `svgs` becomes one SVG endpoint:

```js
export const config = {
  svgs: {
    default: {
      width: 800,
      fontSize: 16,
      rows: [
        { type: "text", text: "Custom static content" },
        { type: "header", name: "Cf-Connecting-Ip", label: "Client IP" },
        { type: "query", name: "user", label: "User" },
        { type: "query", label: "All Query" }
      ]
    }
  }
};
```

Row types:

- `text`: displays static configured text.
- `header`: displays a request header. Header lookup is case-insensitive.
- `query`: displays one query parameter when `name` is set.
- `query` without `name`: expands all query parameters, sorted by key.

## Develop

```powershell
npm install
npm test
npm run dev
```

## Deploy

```powershell
npm run deploy
```
