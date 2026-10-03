// Client-side Mermaid renderer.
// Takes over .mermaid-host containers: decode data-mermaid source, render it, and
// replace the skeleton with a real SVG. Unlike vitepress-plugin-mermaid's Mermaid.vue:
//   1. It does not use Suspense; the skeleton is inlined into the HTML at build time.
//   2. It selects the light or dark theme from documentElement.classList.contains('dark').
//   3. It rescans unrendered hosts after VitePress SPA route changes.
//
// Dynamically import mermaid to avoid loading it during VitePress SSR (mermaid accesses
// document at the top level) and put it in a separate chunk that does not delay initial JS.
let _mermaid = null
async function getMermaid() {
  if (_mermaid) return _mermaid
  const mod = await import('mermaid')
  _mermaid = mod.default
  return _mermaid
}

let booted = false
let renderSeq = 0

async function ensureBoot(isDark) {
  if (booted) return
  const mermaid = await getMermaid()
  mermaid.initialize({
    startOnLoad: false,
    theme: isDark ? 'dark' : 'default',
    securityLevel: 'loose',
    flowchart: { curve: 'basis', useMaxWidth: true },
    sequence: { useMaxWidth: true },
    gantt: { useMaxWidth: true }
  })
  booted = true
}

async function renderOne(host) {
  if (host.dataset.rendered === '1') return
  const code = decodeURIComponent(host.dataset.mermaid || '')
  if (!code) return
  const isDark = document.documentElement.classList.contains('dark')
  await ensureBoot(isDark)
  const mermaid = await getMermaid()
  const id = `mmd-${renderSeq++}`
  try {
    const { svg } = await mermaid.render(id, code)
    const real = host.querySelector('.mermaid-real')
    if (real) {
      real.innerHTML = svg
      host.dataset.rendered = '1'
      host.classList.add('mermaid-rendered')
      // Hide the skeleton.
      const skel = host.querySelector('.mermaid-skeleton')
      if (skel) skel.style.display = 'none'
    }
  } catch (e) {
    // On failure, keep the skeleton, mark the error, and show the source and error as a fallback.
    const errMsg = (e && (e.message || String(e))) || 'unknown error'
    host.classList.add('mermaid-error')
    host.dataset.error = errMsg.slice(0, 500)
    const real = host.querySelector('.mermaid-real')
    if (real) {
      real.innerHTML =
        `<pre class="mermaid-fallback"><code>${escapeHtml(code)}</code>` +
        `<span class="mermaid-fallback-err">⚠ ${escapeHtml(errMsg).slice(0,300)}</span></pre>`
    }
    console.warn('[mermaid] render failed:', e)
  }
}

function escapeHtml(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

async function scan(root = document) {
  const hosts = root.querySelectorAll('.mermaid-host:not([data-rendered="1"])')
  if (!hosts.length) return
  scanning = true
  try {
    // The theme may have changed; initialize again.
    const isDark = document.documentElement.classList.contains('dark')
    if (booted) {
      const mermaid = await getMermaid()
      mermaid.initialize({
        startOnLoad: false,
        theme: isDark ? 'dark' : 'default',
        securityLevel: 'loose'
      })
    }
    await Promise.all([...hosts].map(renderOne))
  } finally {
    scanning = false
  }
}

let darkObserver = null
let scanning = false // Prevent reentrant scans: rendering triggers MutationObserver.

export function setupMermaidRenderer() {
  if (typeof document === 'undefined') return // Skip during SSR.

  // Scan after route changes (VitePress SPA remounts content on every route change).
  let routeTimer = null
  const onRouteChange = () => {
    clearTimeout(routeTimer)
    routeTimer = setTimeout(async () => {
      if (scanning) return
      await scan()
    }, 80)
  }
  window.addEventListener('hashchange', onRouteChange)
  // Fallback: observe added nodes in the main content area after route changes.
  // renderOne also modifies the DOM, so the scanning flag and :not([rendered]) prevent loops.
  const mo = new MutationObserver((muts) => {
    // Trigger only for added nodes; ignore attribute-only changes such as dark-mode toggles.
    const hasAdded = muts.some(m => m.addedNodes.length > 0)
    if (hasAdded) onRouteChange()
  })
  mo.observe(document.body, { childList: true, subtree: true })

  // Rerender when the dark-mode class changes.
  darkObserver = new MutationObserver(() => {
    document
      .querySelectorAll('.mermaid-host[data-rendered="1"]')
      .forEach((host) => {
        host.dataset.rendered = '0'
        const skel = host.querySelector('.mermaid-skeleton')
        if (skel) skel.style.display = ''
        const real = host.querySelector('.mermaid-real')
        if (real) real.innerHTML = ''
        host.classList.remove('mermaid-rendered')
      })
    onRouteChange()
  })
  darkObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class']
  })

  // Perform the initial scan.
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => scan())
  } else {
    scan()
  }
}
