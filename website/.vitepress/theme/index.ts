import { h } from 'vue'
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import './styles/theme.css'
import { setupMermaidRenderer } from './mermaid-renderer'
import HarHero from './components/HarHero.vue'
import HarWaterfall from './components/HarWaterfall.vue'
import FeatureCard from './components/FeatureCard.vue'
import CapabilityTree from './components/CapabilityTree.vue'
import CommandTable from './components/CommandTable.vue'

export default {
  extends: DefaultTheme,
  Layout: () => {
    // Use the default layout and customize its appearance with global styles.
    return h(DefaultTheme.Layout, null, {})
  },
  enhanceApp({ app }) {
    app.component('HarHero', HarHero)
    app.component('HarWaterfall', HarWaterfall)
    app.component('FeatureCard', FeatureCard)
    app.component('CapabilityTree', CapabilityTree)
    app.component('CommandTable', CommandTable)
  },
  setup() {
    // Take over Mermaid skeleton containers on the client: decode source, render, and replace the skeleton.
    setupMermaidRenderer()
  }
} satisfies Theme
