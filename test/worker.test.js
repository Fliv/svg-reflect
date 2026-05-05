import assert from "node:assert/strict";
import test from "node:test";

import {
  handleRequest,
  renderSVG,
  svgProfileFromPath,
  wrapLine
} from "../src/worker.js";

const testConfig = {
  svgs: {
    default: {
      width: 800,
      fontSize: 16,
      rows: [
        { type: "text", text: "Custom static content" },
        { type: "header", name: "Cf-Connecting-Ip", label: "Client IP" },
        { type: "header", name: "User-Agent", label: "User Agent" },
        { type: "query", name: "user", label: "User" },
        { type: "query", label: "All Query" }
      ]
    },
    narrow: {
      width: 170,
      fontSize: 16,
      rows: [{ type: "header", name: "X-Long", label: "Long" }]
    }
  }
};

test("serves configured profile as SVG", async () => {
  const request = new Request("https://worker.example/svg/default.svg?b=2&a=1&a=3&user=alice", {
    headers: {
      "cf-connecting-ip": "203.0.113.10",
      "user-agent": "UnitTest"
    }
  });

  const response = handleRequest(request, testConfig);
  const body = await response.text();

  assert.equal(response.status, 200);
  assert.equal(response.headers.get("Content-Type"), "image/svg+xml; charset=utf-8");
  assert.equal(response.headers.get("Cache-Control"), "no-store");
  assert.match(body, /Custom static content/);
  assert.match(body, /Client IP: 203\.0\.113\.10/);
  assert.match(body, /User Agent: UnitTest/);
  assert.match(body, /User: alice/);
  assert.ok(body.indexOf("All Query.a: 1, 3") < body.indexOf("All Query.b: 2"));
});

test("query row without name expands all query parameters", async () => {
  const request = new Request("https://worker.example/svg/default.svg?z=last&m=mid&m=again&a=first");
  const body = await handleRequest(request, testConfig).text();

  assert.match(body, /All Query.a: first/);
  assert.match(body, /All Query.m: mid, again/);
  assert.match(body, /All Query.z: last/);
  assert.ok(body.indexOf("All Query.a: first") < body.indexOf("All Query.m: mid, again"));
  assert.ok(body.indexOf("All Query.m: mid, again") < body.indexOf("All Query.z: last"));
});

test("unknown or invalid profile returns 404", () => {
  for (const path of ["/svg/missing.svg", "/svg/bad/name.svg", "/svg/bad!.svg", "/svg/default.png"]) {
    const response = handleRequest(new Request(`https://worker.example${path}`), testConfig);
    assert.equal(response.status, 404, path);
  }
});

test("non-GET requests return 405", () => {
  const response = handleRequest(
    new Request("https://worker.example/svg/default.svg", { method: "POST" }),
    testConfig
  );

  assert.equal(response.status, 405);
  assert.equal(response.headers.get("Allow"), "GET");
});

test("SVG escaping protects XML output", () => {
  const svg = renderSVG(
    {
      width: 800,
      fontSize: 16,
      rows: [
        { type: "text", text: `<custom>&"'` },
        { type: "header", name: "X-Danger" },
        { type: "query", name: "q" }
      ]
    },
    new Request("https://worker.example/svg/default.svg?q=%3Cquery%3E%26%22%27", {
      headers: { "X-Danger": `<header>&"'` }
    })
  );

  assert.match(svg, /&lt;custom&gt;&amp;&#34;&#39;/);
  assert.match(svg, /X-Danger: &lt;header&gt;&amp;&#34;&#39;/);
  assert.match(svg, /q: &lt;query&gt;&amp;&#34;&#39;/);
});

test("malformed empty header name renders as an empty value", () => {
  const svg = renderSVG(
    {
      width: 800,
      fontSize: 16,
      rows: [{ type: "header", label: "Empty Header" }]
    },
    new Request("https://worker.example/svg/default.svg")
  );

  assert.match(svg, /Empty Header: /);
});

test("long lines wrap and grow SVG height", async () => {
  const request = new Request("https://worker.example/svg/narrow.svg", {
    headers: { "X-Long": "abcdefghijklmnopqrstuvwxyz0123456789" }
  });
  const body = await handleRequest(request, testConfig).text();

  assert.ok((body.match(/<text /g) ?? []).length >= 4);
  assert.doesNotMatch(body, /abcdefghijklmnopqrstuvwxyz0123456789/);
  assert.doesNotMatch(body, /height="64"/);
});

test("profile parser accepts only safe profile names", () => {
  assert.equal(svgProfileFromPath("/svg/default.svg"), "default");
  assert.equal(svgProfileFromPath("/svg/a_1-b.svg"), "a_1-b");
  assert.equal(svgProfileFromPath("/svg/a/b.svg"), "");
  assert.equal(svgProfileFromPath("/svg/a!.svg"), "");
});

test("wrapLine keeps each line within max chars", () => {
  const lines = wrapLine("Token: abcdefghijklmnopqrstuvwxyz", 12);
  assert.ok(lines.length >= 3);
  for (const line of lines) {
    assert.ok(Array.from(line).length <= 12, line);
  }
});
