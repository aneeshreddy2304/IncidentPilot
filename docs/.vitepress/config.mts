import { defineConfig } from "vitepress";

// Docs site config. The same markdown files are bundled into the SPA's
// in-app docs page (web/src/docs.ts), so pages here must stay plain
// markdown — no VitePress-only syntax in the doc bodies.
export default defineConfig({
  title: "IncidentPilot",
  description:
    "The AI sentinel for your Kubernetes cluster — continuous AI review, alerting, Q&A, and approval-gated remediation.",
  // Served at https://<owner>.github.io/IncidentPilot/
  base: "/IncidentPilot/",
  head: [["link", { rel: "icon", type: "image/png", href: "/IncidentPilot/favicon.png" }]],
  themeConfig: {
    logo: "/logo.png",
    nav: [
      { text: "Docs", link: "/architecture" },
      { text: "Install", link: "/deployment" },
    ],
    sidebar: [
      {
        text: "Documentation",
        items: [
          { text: "Architecture", link: "/architecture" },
          { text: "Deployment", link: "/deployment" },
          { text: "Configuration", link: "/configuration" },
          { text: "AI Agent", link: "/ai-agent" },
          { text: "API Reference", link: "/api" },
          { text: "Security", link: "/security" },
        ],
      },
    ],
    socialLinks: [{ icon: "github", link: "https://github.com/aneeshreddy2304/IncidentPilot" }],
    search: { provider: "local" },
    footer: {
      message: "Released under the Apache-2.0 License.",
      copyright: "Copyright © 2026 Emre Oztoprak and IncidentPilot contributors",
    },
    outline: { level: [2, 3] },
  },
});
