<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'

/**
 * SourceBadge — the canonical provenance pill for an imported story/epic.
 *
 * Reads the authoritative `source` (`manual` | `markdown` | `github_projects`) that the
 * planning importer stamps on every row — it replaces the old `git_provider`-derived
 * "Planned in" heuristic. When a `sourceUrl` deep-link exists (GitHub Projects), the pill
 * is wrapped in an external link back to the origin item.
 *
 * Dumb + prop-driven; no data access.
 */
const props = defineProps<{
  /** Provenance source string from the Story/Epic schema. */
  source: 'manual' | 'markdown' | 'github_projects' | string | null | undefined
  /** Deep-link back to the source item (GitHub Projects). Renders a link when present. */
  sourceUrl?: string | null
}>()

interface SourceMeta {
  label: string
  icon?: string
  severity: 'secondary' | 'info' | 'contrast'
}

const meta = computed<SourceMeta>(() => {
  switch (props.source) {
    case 'markdown':
      return { label: 'Markdown', icon: 'pi pi-file', severity: 'info' }
    case 'github_projects':
      return { label: 'GitHub Projects', icon: 'pi pi-github', severity: 'contrast' }
    default:
      // manual (in-app / seed) and any unknown value: neutral, never linked.
      return { label: 'In-app', icon: 'pi pi-desktop', severity: 'secondary' }
  }
})

/** Only a non-empty source_url turns the badge into a deep-link. */
const hasLink = computed(() => !!props.sourceUrl)
</script>

<template>
  <a
    v-if="hasLink"
    :href="sourceUrl ?? undefined"
    target="_blank"
    rel="noopener"
    class="inline-flex"
    data-testid="source-badge-link"
  >
    <Tag
      :value="meta.label"
      :icon="meta.icon"
      :severity="meta.severity"
      rounded
      :data-source="source ?? 'manual'"
      data-testid="source-badge"
    />
  </a>
  <Tag
    v-else
    :value="meta.label"
    :icon="meta.icon"
    :severity="meta.severity"
    rounded
    :data-source="source ?? 'manual'"
    data-testid="source-badge"
  />
</template>
