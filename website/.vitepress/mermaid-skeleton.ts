// Mermaid skeleton-screen plugin.
// vitepress-plugin-mermaid renders Mermaid code blocks as an empty <div class="mermaid">
// with a hard-coded "Loading..." Suspense fallback. The page remains blank until the
// client-side renderer loads, making the documentation site appear empty.
//
// This plugin intercepts Mermaid fences in markdown-it and emits a container with a placeholder:
//   <div class="mermaid-host" data-mermaid="<source>">
//     <div class="mermaid-skeleton" aria-hidden="true">…skeleton placeholder…</div>
//   </div>
// The skeleton includes visual structure (a pulsing frame, icon, and category label),
// so the initial page is not blank. The client renderer replaces it with the SVG.
//
// This coexists with vitepress-plugin-mermaid: because this plugin returns first,
// the latter's fence handler is not triggered. The mermaid package is still used
// for client-side rendering; only the placeholder is improved.

const CATEGORY_LABELS = {
  flowchart: 'Flowchart',
  'sequenceDiagram': 'Sequence Diagram',
  'stateDiagram': 'State Diagram',
  'classDiagram': 'Class Diagram',
  'erDiagram': 'ER Diagram',
  gantt: 'Gantt Chart',
  pie: 'Pie Chart',
  mindmap: 'Mind Map',
  journey: 'User Journey',
  gitGraph: 'Git Graph'
}

function detectCategory(code) {
  const c = code.trim()
  for (const key of Object.keys(CATEGORY_LABELS)) {
    if (c.startsWith(key) || c.includes('\n' + key)) return key
  }
  // Treat flowchart and graph as flowcharts too.
  if (/^(flowchart|graph)\s/.test(c)) return 'flowchart'
  return null
}

export function mermaidSkeletonPlugin(md) {
  const original = md.renderer.rules.fence
  md.renderer.rules.fence = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    if (token.info.trim() === 'mermaid') {
      const code = token.content
      const cat = detectCategory(code)
      const label = cat ? CATEGORY_LABELS[cat] : 'Diagram'
      // URL-encode the source into a data attribute for the client script to decode for mermaid.render.
      const encoded = encodeURIComponent(code)
      return (
        `<div class="mermaid-host" data-mermaid="${encoded}" data-cat="${cat || ''}">` +
          `<div class="mermaid-skeleton" aria-hidden="true">` +
            `<div class="mermaid-skeleton-bar"></div>` +
            `<div class="mermaid-skeleton-body">` +
              `<span class="mermaid-skeleton-icon">◷</span>` +
              `<span class="mermaid-skeleton-label">${label}</span>` +
            `</div>` +
            `<div class="mermaid-skeleton-shimmer"></div>` +
          `</div>` +
          `<div class="mermaid-real" role="img"></div>` +
        `</div>`
      )
    }
    return original.call(this, tokens, idx, options, env, self)
  }
}
