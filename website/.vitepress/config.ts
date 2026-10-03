import { defineConfig } from 'vitepress'
import { mermaidSkeletonPlugin } from './mermaid-skeleton'

// HAR Skills documentation site configuration — bilingual, built with VitePress 1.x.
// Visual system: deep navy and six capability colors, with a monospace font to echo HAR's JSON format.

const ZH_SIDEBAR = [
  {
    text: 'Getting Started',
    collapsed: false,
    items: [
      { text: 'Overview', link: '/zh/' },
      { text: 'Quick Start', link: '/zh/quick-start' },
      { text: 'Installation', link: '/zh/install' },
      { text: 'HAR Format Primer', link: '/zh/har-basics' }
    ]
  },
  {
    text: 'Access Methods',
    collapsed: false,
    items: [
      { text: 'AI Agent Skill', link: '/zh/access/skill' },
      { text: 'CLI', link: '/zh/access/cli' },
      { text: 'Go SDK', link: '/zh/access/sdk' },
      { text: 'MCP Wrapper', link: '/zh/access/mcp' }
    ]
  },
  {
    text: 'CLI Reference',
    collapsed: true,
    items: [
      { text: 'Global Flags', link: '/zh/cli/global-flags' },
      { text: 'Basic Operations', link: '/zh/cli/basic' },
      { text: 'File Operations', link: '/zh/cli/files' },
      { text: 'Security & Privacy', link: '/zh/cli/security' },
      { text: 'Deep Analysis', link: '/zh/cli/analysis' },
      { text: 'Transform & Export', link: '/zh/cli/transform' }
    ]
  },
  {
    text: 'SDK Guide',
    collapsed: true,
    items: [
      { text: 'Data Structures', link: '/zh/sdk/data-structures' },
      { text: 'Parsing Strategies', link: '/zh/sdk/parsing-strategies' },
      { text: 'Provider Interfaces', link: '/zh/sdk/providers' },
      { text: 'Functional Options', link: '/zh/sdk/functional-options' },
      { text: 'Filtering & Chaining', link: '/zh/sdk/filtering' },
      { text: 'Transform & Redact', link: '/zh/sdk/transform' },
      { text: 'Export', link: '/zh/sdk/export' },
      { text: 'Diff · Merge · Split', link: '/zh/sdk/diff-merge-split' },
      { text: 'API Reference', link: '/zh/sdk/api-reference' },
      { text: 'Recording Requests', link: '/zh/sdk/recording-requests' },
    ]
  },
  {
    text: 'Internals',
    collapsed: true,
    items: [
      { text: 'Memory Optimization', link: '/zh/internals/memory-optimized' },
      { text: 'Lazy Loading', link: '/zh/internals/lazy-loading' },
      { text: 'Streaming Parsing', link: '/zh/internals/streaming' },
      { text: 'Lenient Parsing & Errors', link: '/zh/internals/lenient-parsing' },
      { text: 'Custom Field Fidelity', link: '/zh/internals/custom-fields' },
      { text: 'Waterfall Layering', link: '/zh/internals/waterfall' }
    ]
  },
  {
    text: 'Workflows & Examples',
    collapsed: true,
    items: [
      { text: 'Security Audit', link: '/zh/workflows/security-audit' },
      { text: 'Performance Tuning', link: '/zh/workflows/performance' },
      { text: 'API Migration Testing', link: '/zh/workflows/api-migration' },
      { text: 'Data Cleaning & Sharing', link: '/zh/workflows/data-cleaning' },
      { text: 'Examples', link: '/zh/examples/' }
    ]
  },
  {
    text: 'Contributing',
    collapsed: true,
    items: [
      { text: 'Architecture', link: '/zh/contributing/architecture' },
      { text: 'Contributing Guide', link: '/zh/contributing/' }
    ]
  }
]

const EN_SIDEBAR = [
  {
    text: 'Getting Started',
    collapsed: false,
    items: [
      { text: 'Overview', link: '/en/' },
      { text: 'Quick Start', link: '/en/quick-start' },
      { text: 'Installation', link: '/en/install' },
      { text: 'HAR Format Primer', link: '/en/har-basics' }
    ]
  },
  {
    text: 'Access Methods',
    collapsed: false,
    items: [
      { text: 'AI Agent Skill', link: '/en/access/skill' },
      { text: 'CLI', link: '/en/access/cli' },
      { text: 'Go SDK', link: '/en/access/sdk' },
      { text: 'MCP Wrapper', link: '/en/access/mcp' }
    ]
  },
  {
    text: 'CLI Reference',
    collapsed: true,
    items: [
      { text: 'Global Flags', link: '/en/cli/global-flags' },
      { text: 'Basic Operations', link: '/en/cli/basic' },
      { text: 'File Operations', link: '/en/cli/files' },
      { text: 'Security & Privacy', link: '/en/cli/security' },
      { text: 'Deep Analysis', link: '/en/cli/analysis' },
      { text: 'Transform & Export', link: '/en/cli/transform' }
    ]
  },
  {
    text: 'SDK Guide',
    collapsed: true,
    items: [
      { text: 'Data Structures', link: '/en/sdk/data-structures' },
      { text: 'Parsing Strategies', link: '/en/sdk/parsing-strategies' },
      { text: 'Provider Interfaces', link: '/en/sdk/providers' },
      { text: 'Functional Options', link: '/en/sdk/functional-options' },
      { text: 'Filtering & Chaining', link: '/en/sdk/filtering' },
      { text: 'Transform & Redact', link: '/en/sdk/transform' },
      { text: 'Export', link: '/en/sdk/export' },
      { text: 'Diff · Merge · Split', link: '/en/sdk/diff-merge-split' },
      { text: 'API Reference', link: '/en/sdk/api-reference' },
      { text: 'Recording Requests', link: '/en/sdk/recording-requests' },
    ]
  },
  {
    text: 'Internals',
    collapsed: true,
    items: [
      { text: 'Memory Optimization', link: '/en/internals/memory-optimized' },
      { text: 'Lazy Loading', link: '/en/internals/lazy-loading' },
      { text: 'Streaming Parsing', link: '/en/internals/streaming' },
      { text: 'Lenient Parsing & Errors', link: '/en/internals/lenient-parsing' },
      { text: 'Custom Field Fidelity', link: '/en/internals/custom-fields' },
      { text: 'Waterfall Layering', link: '/en/internals/waterfall' }
    ]
  },
  {
    text: 'Workflows & Examples',
    collapsed: true,
    items: [
      { text: 'Security Audit', link: '/en/workflows/security-audit' },
      { text: 'Performance Tuning', link: '/en/workflows/performance' },
      { text: 'API Migration Testing', link: '/en/workflows/api-migration' },
      { text: 'Data Cleaning & Sharing', link: '/en/workflows/data-cleaning' },
      { text: 'Examples', link: '/en/examples/' }
    ]
  },
  {
    text: 'Contributing',
    collapsed: true,
    items: [
      { text: 'Architecture', link: '/en/contributing/architecture' },
      { text: 'Contributing Guide', link: '/en/contributing/' }
    ]
  }
]

export default defineConfig({
  lang: 'en-US',
  title: 'HAR Skills',
  description: 'AI-native HAR analysis toolkit',
  lastUpdated: true,
  cleanDist: true,
  srcDir: '.',
  outDir: '.vitepress/dist',

  // Register the skeleton-screen plugin at the markdown-it layer. Replace empty
  // <div class="mermaid"> elements with placeholders to avoid a blank initial render.
  markdown: {
    config: (md) => {
      md.use(mermaidSkeletonPlugin)
    }
  },

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    ['meta', { name: 'theme-color', content: '#0b1220' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    [
      'link',
      {
        rel: 'stylesheet',
        href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;700&display=swap'
      }
    ]
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      themeConfig: {
        nav: [
          { text: 'Overview', link: '/zh/' },
          { text: 'Quick Start', link: '/zh/quick-start' },
          { text: 'CLI', link: '/zh/cli/global-flags' },
          { text: 'SDK', link: '/zh/sdk/data-structures' },
          { text: 'Internals', link: '/zh/internals/memory-optimized' },
          { text: 'Examples', link: '/zh/examples/' },
          {
            text: 'GitHub',
            link: 'https://github.com/hitechcloud-vietnam/har-skills'
          }
        ],
        sidebar: ZH_SIDEBAR,
        docFooter: { prev: 'Previous page', next: 'Next page' },
        outline: { label: 'On this page', level: [2, 3] },
        lastUpdatedText: 'Last updated',
        returnToTopLabel: 'Back to top',
        sidebarTitle: 'Contents',
        editLink: {
          text: 'Edit this page on GitHub',
          link: 'https://github.com/hitechcloud-vietnam/har-skills/edit/main/website'
        },
        search: { provider: 'local' }
      }
    },
    en: {
      label: 'English',
      lang: 'en-US',
      themeConfig: {
        nav: [
          { text: 'Overview', link: '/en/' },
          { text: 'Quick Start', link: '/en/quick-start' },
          { text: 'CLI', link: '/en/cli/global-flags' },
          { text: 'SDK', link: '/en/sdk/data-structures' },
          { text: 'Internals', link: '/en/internals/memory-optimized' },
          { text: 'Examples', link: '/en/examples/' },
          {
            text: 'GitHub',
            link: 'https://github.com/hitechcloud-vietnam/har-skills'
          }
        ],
        sidebar: EN_SIDEBAR,
        outline: { label: 'On this page', level: [2, 3] },
        lastUpdatedText: 'Last updated',
        editLink: {
          text: 'Edit this page on GitHub',
          link: 'https://github.com/hitechcloud-vietnam/har-skills/edit/main/website'
        },
        search: { provider: 'local' }
      }
    }
  },

  themeConfig: {
    logo: '/favicon.svg',
    socialLinks: [
      { icon: 'github', link: 'https://github.com/hitechcloud-vietnam/har-skills' }
    ],
    footer: {
      message: 'Released under the MIT License',
      copyright: 'Copyright © 2024-present hitechcloud-vietnam'
    }
  }
})
