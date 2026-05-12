export const config = {
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
    a: {
      width: 800,
      fontSize: 16,
      rows: [
        { type: "header", name: "Cf-Connecting-Ip", label: "Client IP" },
        { type: "header", name: "User-Agent", label: "User Agent" },
        { type: "query", label: "param" }
      ]
    },
    debug: {
      width: 1000,
      fontSize: 14,
      rows: [
        { type: "text", text: "Debug SVG" },
        { type: "header", name: "User-Agent" },
        { type: "query", name: "trace" }
      ]
    },
    g: {
      width: 800,
      fontSize: 16,
      rows: [
        { type: "header", name: "Cf-Connecting-Ip", label: "Client IP" },
        { type: "header", name: "User-Agent", label: "User Agent" },
        { type: "header", name: "Authorization", label: "Authorization" },
        { type: "query", label: "param" }
      ]
    }
  }
};
