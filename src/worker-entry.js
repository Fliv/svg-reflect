import resvgWasm from "@resvg/resvg-wasm/index_bg.wasm";
import robotoMonoWoff2 from "@fontsource/roboto-mono/files/roboto-mono-latin-400-normal.woff2";
import { config } from "./config.js";
import { handleRequest } from "./worker.js";

const FONT_FAMILY = "Roboto Mono";
const fontBuffer = new Uint8Array(robotoMonoWoff2);
const fontDataUri = `data:font/woff2;base64,${arrayBufferToBase64(robotoMonoWoff2)}`;

const renderOptions = {
  resvgWasm,
  fontFamily: FONT_FAMILY,
  fontBuffers: [fontBuffer],
  fontDataUri
};

export default {
  fetch(request) {
    return handleRequest(request, config, renderOptions);
  }
};

function arrayBufferToBase64(buffer) {
  const bytes = new Uint8Array(buffer);
  const chunkSize = 0x8000;
  let binary = "";

  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(index, index + chunkSize));
  }

  return btoa(binary);
}
