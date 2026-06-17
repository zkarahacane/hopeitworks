<script setup lang="ts">
import { inject, ref, computed, watch, onMounted, type Ref } from 'vue'
import { formatDate } from '@/utils/formatDate'
import { apiClient } from '@/api/client'
import type { Project } from '@/stores/projects'

const project = inject<Ref<Project | null>>('project')

const p = computed(() => project?.value ?? null)

const providerIcons: Record<string, string> = {
  github: 'pi pi-github',
  gitlab: 'pi pi-gitlab',
  gitea: 'pi pi-server',
  bitbucket: 'pi pi-code',
}

const storyCount = ref<number | null>(null)
const runCount = ref<number | null>(null)

async function fetchCounts() {
  if (!p.value?.id) return

  const [storiesRes, runsRes] = await Promise.all([
    apiClient.GET('/projects/{projectId}/stories', {
      params: { path: { projectId: p.value.id }, query: { per_page: 1, page: 1 } },
    }),
    apiClient.GET('/projects/{projectId}/runs', {
      params: { path: { projectId: p.value.id }, query: { per_page: 1, page: 1 } },
    }),
  ])

  // The API response is a union — cast to the paginated shape to access pagination
  const storiesData = storiesRes.data as { pagination?: { total?: number } } | undefined
  const runsData = runsRes.data as { pagination?: { total?: number } } | undefined
  storyCount.value = storiesData?.pagination?.total ?? null
  runCount.value = runsData?.pagination?.total ?? null
}

onMounted(fetchCounts)

watch(
  () => p.value?.id,
  (id) => {
    if (id) fetchCounts()
  },
)
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <template v-if="p">
      <!-- 2-column grid -->
      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Left: Project info -->
        <div
          data-testid="project-overview-card"
          style="
            background: var(--surface-raised);
            border: 1px solid var(--surface-border);
            border-radius: 0.5rem;
            padding: 1.5rem;
          "
        >
          <p
            style="
              font-size: 0.75rem;
              font-weight: 600;
              text-transform: uppercase;
              letter-spacing: 0.05em;
              color: var(--p-text-muted-color);
              margin-bottom: 1rem;
            "
          >
            Project info
          </p>

          <dl class="flex flex-col gap-4">
            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Name</dt>
              <dd style="font-weight: 500">{{ p.name }}</dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Description</dt>
              <dd>{{ p.description || '—' }}</dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Repository</dt>
              <dd>
                <a
                  v-if="p.repo_url"
                  :href="p.repo_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  style="color: var(--p-primary-color); text-decoration: underline; word-break: break-all"
                >{{ p.repo_url }}</a>
                <span v-else>—</span>
              </dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Git provider</dt>
              <dd class="flex items-center gap-2">
                <template v-if="p.git_provider">
                  <i
                    :class="providerIcons[p.git_provider.toLowerCase()] ?? 'pi pi-code'"
                    style="font-size: 0.875rem"
                  />
                  <span>{{ p.git_provider }}</span>
                </template>
                <span v-else>—</span>
              </dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Default model</dt>
              <dd>
                <span v-if="p.default_model" style="font-family: monospace; font-size: 0.875rem">{{ p.default_model }}</span>
                <span v-else>—</span>
              </dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Created</dt>
              <dd>{{ formatDate(p.created_at) }}</dd>
            </div>
          </dl>
        </div>

        <!-- Right: Runtime panel -->
        <div
          style="
            background: var(--surface-raised);
            border: 1px solid var(--surface-border);
            border-radius: 0.5rem;
            padding: 1.5rem;
          "
        >
          <p
            style="
              font-size: 0.75rem;
              font-weight: 600;
              text-transform: uppercase;
              letter-spacing: 0.05em;
              color: var(--p-text-muted-color);
              margin-bottom: 1rem;
            "
          >
            Runtime
          </p>

          <dl class="flex flex-col gap-4">
            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Agent runtime</dt>
              <dd class="flex items-center gap-2">
                <template v-if="p.agent_runtime">
                  <i class="pi pi-box" style="font-size: 0.875rem" />
                  <span style="font-family: monospace; font-size: 0.875rem">{{ p.agent_runtime }}</span>
                </template>
                <span v-else>—</span>
              </dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Max parallel</dt>
              <dd>—</dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Isolation</dt>
              <dd>—</dd>
            </div>

            <div>
              <dt style="font-size: 0.75rem; color: var(--p-text-muted-color); margin-bottom: 0.25rem">Base branch</dt>
              <dd>main</dd>
            </div>
          </dl>
        </div>
      </div>

      <!-- Stats bar -->
      <div class="flex items-center gap-3">
        <span
          style="
            background: var(--surface-overlay);
            border-radius: 0.25rem;
            padding: 0.25rem 0.75rem;
            font-size: 0.875rem;
            color: var(--p-text-color);
          "
        >
          {{ storyCount !== null ? storyCount + ' stories' : '— stories' }}
        </span>
        <span
          style="
            background: var(--surface-overlay);
            border-radius: 0.25rem;
            padding: 0.25rem 0.75rem;
            font-size: 0.875rem;
            color: var(--p-text-color);
          "
        >
          {{ runCount !== null ? runCount + ' runs' : '— runs' }}
        </span>
      </div>
    </template>

    <p v-else style="color: var(--p-text-muted-color)">Loading project details…</p>
  </div>
</template>
