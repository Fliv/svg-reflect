import { Resvg, initWasm } from "@resvg/resvg-wasm";
import { config as defaultConfig } from "./config.js";

const DEFAULT_WIDTH = 800;
const DEFAULT_FONT_SIZE = 16;
const SVG_PADDING_X = 24;
const SVG_PADDING_Y = 18;
const MONOSPACE_CHAR_WIDTH_NUMERATOR = 6;
const MONOSPACE_CHAR_WIDTH_DENOMINATOR = 10;
const CONTINUATION_PREFIX = "  ";
const PROFILE_PATTERN = /^[A-Za-z0-9_-]+$/;
const DEFAULT_FONT_FAMILY = "ui-monospace, SFMono-Regular, Consolas, Liberation Mono, monospace";

let resvgInitPromise = null;

export default {
  fetch(request) {
    return handleRequest(request, defaultConfig);
  }
};

export async function handleRequest(request, config = defaultConfig, options = {}) {
  if (request.method !== "GET") {
    return new Response("method not allowed", {
      status: 405,
      headers: { Allow: "GET" }
    });
  }

  const url = new URL(request.url);
  const route = imageRouteFromPath(url.pathname);
  if (!route) {
    return new Response("not found", { status: 404 });
  }

  const svgConfig = config?.svgs?.[route.profile];
  if (!svgConfig) {
    return new Response("not found", { status: 404 });
  }

  const svg = renderSVG(svgConfig, request, url, options);
  if (route.format === "png") {
    return new Response(await svgToPNG(svg, options), {
      status: 200,
      headers: {
        "Content-Type": "image/png",
        "Cache-Control": "no-store"
      }
    });
  }

  return new Response(svg, {
    status: 200,
    headers: {
      "Content-Type": "image/svg+xml; charset=utf-8",
      "Cache-Control": "no-store"
    }
  });
}

export function imageRouteFromPath(pathname) {
  const match = pathname.match(/^\/svg\/([^/]+)\.(svg|png)$/);
  if (!match) {
    return null;
  }
  const profile = match[1];
  return PROFILE_PATTERN.test(profile) ? { profile, format: match[2] } : null;
}

export function svgProfileFromPath(pathname) {
  const match = pathname.match(/^\/svg\/([^/]+)\.svg$/);
  if (!match) {
    return "";
  }
  const profile = match[1];
  return PROFILE_PATTERN.test(profile) ? profile : "";
}

export function renderSVG(svgConfig, request, url = new URL(request.url), options = {}) {
  const width = positiveInt(svgConfig.width, DEFAULT_WIDTH);
  const fontSize = positiveInt(svgConfig.fontSize ?? svgConfig.font_size, DEFAULT_FONT_SIZE);
  const fontFamily = String(options.fontFamily ?? DEFAULT_FONT_FAMILY);
  const rows = Array.isArray(svgConfig.rows) ? svgConfig.rows : [];
  const lines = wrapRenderedLines(renderLines(rows, request, url), width, fontSize);
  const lineHeight = Math.max(fontSize + Math.floor(fontSize / 2), fontSize + 6);
  const height = Math.max(64, SVG_PADDING_Y * 2 + lineHeight * lines.length);
  const fontDefinition = renderFontDefinition(options);

  const textLines = lines
    .map((line, index) => {
      const y = SVG_PADDING_Y + fontSize + index * lineHeight;
      return `    <text x="${SVG_PADDING_X}" y="${y}">${escapeXML(line)}</text>`;
    })
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
${fontDefinition}  <rect width="100%" height="100%" fill="#ffffff"/>
  <g font-family="${escapeXML(fontFamily)}" font-size="${fontSize}" fill="#111827">
${textLines}
  </g>
</svg>
`;
}

export async function svgToPNG(svg, options = {}) {
  await ensureResvgInitialized(options.resvgWasm);

  const fontBuffers = normalizeFontBuffers(options.fontBuffers);
  const resvg = new Resvg(svg, {
    font:
      fontBuffers.length > 0
        ? {
            fontBuffers,
            defaultFontFamily: options.fontFamily,
            loadSystemFonts: false
          }
        : {
            defaultFontFamily: options.fontFamily,
            loadSystemFonts: true
          },
    textRendering: 1
  });

  try {
    return resvg.render().asPng();
  } finally {
    resvg.free();
  }
}

export function renderLines(rows, request, url) {
  const lines = [];
  for (const row of rows) {
    if (!row || typeof row !== "object") {
      continue;
    }

    if (row.type === "text") {
      lines.push(String(row.text ?? ""));
      continue;
    }

    if (row.type === "header") {
      const name = String(row.name ?? "");
      const label = labelOrName(row.label, name);
      lines.push(formatKV(label, headerValue(request.headers, name)));
      continue;
    }

    if (row.type === "query") {
      const name = String(row.name ?? "");
      if (name !== "") {
        const label = labelOrName(row.label, name);
        lines.push(formatKV(label, url.searchParams.getAll(name).join(", ")));
        continue;
      }
      lines.push(...renderAllQueryLines(row.label, url.searchParams));
    }
  }
  return lines;
}

export function renderAllQueryLines(label, searchParams) {
  const keys = Array.from(new Set(searchParams.keys())).sort();
  if (keys.length === 0) {
    return [formatKV(label ? String(label) : "query", "")];
  }

  return keys.map((key) => {
    const displayName = label ? `${label}.${key}` : key;
    return formatKV(displayName, searchParams.getAll(key).join(", "));
  });
}

export function wrapRenderedLines(lines, width, fontSize) {
  const maxChars = maxLineChars(width, fontSize);
  return lines.flatMap((line) => wrapLine(String(line), maxChars));
}

export function maxLineChars(width, fontSize) {
  const contentWidth = Math.max(0, positiveInt(width, DEFAULT_WIDTH) - SVG_PADDING_X * 2);
  const normalizedFontSize = positiveInt(fontSize, DEFAULT_FONT_SIZE);
  return Math.max(
    1,
    Math.floor(
      (contentWidth * MONOSPACE_CHAR_WIDTH_DENOMINATOR) /
        (normalizedFontSize * MONOSPACE_CHAR_WIDTH_NUMERATOR)
    )
  );
}

export function wrapLine(line, maxChars) {
  const limit = Math.max(1, maxChars);
  let remaining = Array.from(line);
  if (remaining.length <= limit) {
    return [line];
  }

  const lines = [];
  let prefix = "";
  while (remaining.length > 0) {
    let available = limit - Array.from(prefix).length;
    if (available < 1) {
      available = limit;
      prefix = "";
    }

    if (remaining.length <= available) {
      lines.push(prefix + remaining.join(""));
      break;
    }

    const splitAt = bestWrapSplit(remaining, available);
    const chunk = remaining.slice(0, splitAt).join("").replace(/\s+$/u, "");
    lines.push(prefix + chunk);
    remaining = trimLeftSpaces(remaining.slice(splitAt));
    prefix = CONTINUATION_PREFIX;
  }

  return lines.length > 0 ? lines : [""];
}

function renderFontDefinition(options) {
  if (!options.fontDataUri || !options.fontFamily) {
    return "";
  }

  return `  <defs>
    <style><![CDATA[
      @font-face {
        font-family: '${cssString(options.fontFamily)}';
        src: url(${options.fontDataUri}) format('woff2');
        font-weight: 400;
        font-style: normal;
      }
    ]]></style>
  </defs>
`;
}

async function ensureResvgInitialized(resvgWasm) {
  if (!resvgWasm) {
    throw new Error("resvg wasm input is required to render PNG");
  }

  if (!resvgInitPromise) {
    resvgInitPromise = initWasm(resvgWasm).catch((error) => {
      resvgInitPromise = null;
      throw error;
    });
  }
  await resvgInitPromise;
}

function normalizeFontBuffers(fontBuffers) {
  if (!Array.isArray(fontBuffers)) {
    return [];
  }

  return fontBuffers.map((buffer) => {
    if (buffer instanceof Uint8Array) {
      return buffer;
    }
    if (buffer instanceof ArrayBuffer) {
      return new Uint8Array(buffer);
    }
    if (ArrayBuffer.isView(buffer)) {
      return new Uint8Array(buffer.buffer, buffer.byteOffset, buffer.byteLength);
    }
    return new Uint8Array(buffer);
  });
}

function bestWrapSplit(chars, limit) {
  if (limit >= chars.length) {
    return chars.length;
  }
  for (let index = limit - 1; index > 0; index -= 1) {
    if (isPreferredWrapChar(chars[index])) {
      return index + 1;
    }
  }
  return limit;
}

function isPreferredWrapChar(char) {
  return char === " " || char === "," || char === ";" || char === "&" || char === "?" || char === "/" || char === "\\";
}

function trimLeftSpaces(chars) {
  let index = 0;
  while (index < chars.length && chars[index] === " ") {
    index += 1;
  }
  return chars.slice(index);
}

function labelOrName(label, name) {
  return label ? String(label) : name;
}

function formatKV(label, value) {
  return `${label}: ${value}`;
}

function headerValue(headers, name) {
  if (name === "") {
    return "";
  }
  try {
    return headers.get(name) ?? "";
  } catch {
    return "";
  }
}

function positiveInt(value, fallback) {
  const parsed = Number(value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback;
}

function escapeXML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&#34;")
    .replaceAll("'", "&#39;");
}

function cssString(value) {
  return String(value).replace(/\\/g, "\\\\").replace(/'/g, "\\'");
}
