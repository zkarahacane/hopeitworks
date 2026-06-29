<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import AutoComplete from 'primevue/autocomplete'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth'
import {
  usePlanningConnector,
  type PlanningConnector,
  type PlanningStatusOption,
} from '@/composables/usePlanningConnector'
import { useGitConnection, type GitConnectionStatus } from './useGitConnection'
import type { Project } from '@/stores/projects'

const props = defineProps<{
  project: Project
}>()

const authStore = useAuthStore()
const toast = useToast()

// Authorization: same guard as GitConnectionCard — owner or global admin.
const canManage = computed(() => {
  const u = authStore.user
  if (!u) return false
  return u.role === 'admin' || props.project.owner_id === u.id
})

const { fetchConnector, saveConnector, fetchStatusOptions } = usePlanningConnector()
const { status: gitStatus } = useGitConnection()

// ── State ─────────────────────────────────────────────────────────────────────

/** Persisted connector (null = not configured yet). */
const connector = ref<PlanningConnector | null>(null)
/** Git connection advisory status (to warn when write-back won't work). */
const gitConn = ref<GitConnectionStatus | null>(null)

const gitConnected = computed(() => gitConn.value?.status === 'connected')

// Form fields — synced from connector on load, edited locally.
const projectUrl = ref('')
const statusField = ref('Status')
const doneOptions = ref<string[]>([])
const epicIssueType = ref('Epic')
const writebackEnabled = ref(false)
const postRunComment = ref(false)

// Status mapping: internal key → external option id (string | null).
const mappingBacklog = ref<string | null>(null)
const mappingRunning = ref<string | null>(null)
const mappingDone = ref<string | null>(null)
const mappingFailed = ref<string | null>(null)

/** Live options fetched from the tracker (for the mapping selects). */
const statusOptions = ref<PlanningStatusOption[]>([])
/** True when the status-options fetch returned at least one result. */
const optionsLoaded = ref(false)

// ── Bootstrap ─────────────────────────────────────────────────────────────────

function populateForm(c: PlanningConnector | null) {
  if (!c) return
  projectUrl.value = c.project_url ?? ''
  statusField.value = c.status_field ?? 'Status'
  doneOptions.value = c.done_options ?? []
  epicIssueType.value = c.epic_issue_type ?? 'Epic'
  writebackEnabled.value = c.writeback_enabled ?? false
  postRunComment.value = c.post_run_comment ?? false
  mappingBacklog.value = c.status_mapping?.backlog ?? null
  mappingRunning.value = c.status_mapping?.running ?? null
  mappingDone.value = c.status_mapping?.done ?? null
  mappingFailed.value = c.status_mapping?.failed ?? null
}

async function refresh() {
  const [connResult, gitResult] = await Promise.all([
    fetchConnector.execute(props.project.id),
    gitStatus.execute(props.project.id),
  ])
  // fetchConnector returns null | PlanningConnector. The useAsyncAction wraps null as valid data.
  connector.value = connResult ?? null
  gitConn.value = gitResult ?? null
  populateForm(connector.value)
}

onMounted(() => {
  if (canManage.value) refresh()
})

// ── Status options (mapping) ──────────────────────────────────────────────────

async function handleLoadOptions() {
  const result = await fetchStatusOptions.execute(props.project.id, {
    project_url: projectUrl.value.trim() || undefined,
    status_field: statusField.value.trim() || undefined,
  })
  if (result) {
    statusOptions.value = result.options
    optionsLoaded.value = true
  }
}

/**
 * Auto-fill mapping by convention (case-insensitive keyword matching).
 * Leaves unrecognised statuses empty so the user can map them manually.
 */
function handleAutoFill() {
  if (!statusOptions.value.length) return

  function findOption(keywords: string[]): string | null {
    const opt = statusOptions.value.find((o) =>
      keywords.some((kw) => o.name.toLowerCase().includes(kw)),
    )
    return opt?.id ?? null
  }

  mappingBacklog.value = findOption(['backlog', 'todo', 'to do', 'open', 'new'])
  mappingRunning.value = findOption(['progress', 'doing', 'in-progress', 'started', 'active'])
  mappingDone.value = findOption(['done', 'closed', 'complete', 'finished', 'merged'])
  // `failed` rarely has a direct equivalent — leave it null if no obvious match.
  mappingFailed.value = findOption(['failed', 'blocked', 'error', 'rejected', 'cancelled'])
}

// ── Computed helpers for select options ──────────────────────────────────────

/**
 * Option list for the mapping selects: null entry (= "not mapped") + live options.
 * Using object shape so Select renders both the placeholder and the real options.
 */
const mappingSelectOptions = computed(() => [
  { id: null, name: '— not mapped —' },
  ...statusOptions.value,
])

// ── Save ─────────────────────────────────────────────────────────────────────

async function handleSave() {
  const result = await saveConnector.execute(props.project.id, {
    source: 'github_projects',
    project_url: projectUrl.value.trim() || undefined,
    status_field: statusField.value.trim() || 'Status',
    done_options: doneOptions.value,
    epic_issue_type: epicIssueType.value.trim() || 'Epic',
    status_mapping: {
      backlog: mappingBacklog.value ?? null,
      running: mappingRunning.value ?? null,
      done: mappingDone.value ?? null,
      failed: mappingFailed.value ?? null,
    },
    writeback_enabled: writebackEnabled.value,
    post_run_comment: postRunComment.value,
  })
  if (result) {
    connector.value = result
    toast.add({ severity: 'success', summary: 'Tracker connector saved', life: 3000 })
  }
}

// Reset option list whenever the project_url or status_field changes so stale
// mapping options are not silently reused.
watch([projectUrl, statusField], () => {
  statusOptions.value = []
  optionsLoaded.value = false
})

const isBusy = computed(
  () =>
    fetchConnector.isLoading.value ||
    saveConnector.isLoading.value ||
    fetchStatusOptions.isLoading.value ||
    gitStatus.isLoading.value,
)

const saveError = computed(() => saveConnector.error.value?.message ?? null)
const optionsError = computed(() => fetchStatusOptions.error.value?.message ?? null)
</script>

<template>
  <section
    data-testid="planning-connector-card"
    class="flex flex-col gap-4"
    style="
      background: var(--surface-raised);
      border: 1px solid var(--surface-border);
      border-radius: 0.5rem;
      padding: 1.5rem;
    "
  >
    <!-- Header -->
    <div class="flex flex-col gap-1">
      <h3 class="text-sm font-semibold">Tracker &amp; sync</h3>
      <p class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
        Configure the GitHub Projects v2 board linked to this project. Once saved, the import
        dialog pre-fills these values and write-back can push internal status transitions back
        to the tracker.
      </p>
    </div>

    <!-- Loading skeleton -->
    <template v-if="fetchConnector.isLoading.value">
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
      <Skeleton height="8rem" />
    </template>

    <!-- Read-only path for non-owner / non-admin -->
    <Message
      v-else-if="!canManage"
      severity="secondary"
      :closable="false"
      data-testid="planning-connector-readonly"
    >
      Configuring the tracker connector is restricted to the project owner or a global admin.
    </Message>

    <!-- Owner / admin form -->
    <template v-else>
      <!-- Empty state hint -->
      <Message
        v-if="!fetchConnector.isLoading.value && !connector"
        severity="info"
        :closable="false"
        data-testid="planning-connector-empty"
      >
        No connector configured yet. Fill in the form below to link this project to a GitHub
        Projects v2 board.
      </Message>

      <!-- Git connection warning (write-back prerequisite) -->
      <Message
        v-if="!gitConnected && !gitStatus.isLoading.value"
        severity="warn"
        :closable="false"
        data-testid="planning-connector-no-git"
      >
        <span>This project is not connected to GitHub. </span>
        <strong>Write-back will not work</strong> until a git connection is configured above.
      </Message>

      <!-- ── Project URL ───────────────────────────────────────────────────── -->
      <div class="flex flex-col gap-1">
        <label for="pc-project-url" class="text-sm font-medium">GitHub Projects URL</label>
        <InputText
          id="pc-project-url"
          v-model="projectUrl"
          placeholder="https://github.com/orgs/<org>/projects/<n>"
          fluid
          :disabled="isBusy"
          data-testid="pc-project-url"
        />
        <small :style="{ color: 'var(--p-text-muted-color)' }">
          The URL of the GitHub Projects v2 board to link to this project.
        </small>
      </div>

      <!-- ── Status field + Epic issue type (row) ─────────────────────────── -->
      <div class="flex flex-wrap gap-4">
        <div class="flex flex-col gap-1 flex-1 min-w-40">
          <label for="pc-status-field" class="text-sm font-medium">Status field</label>
          <InputText
            id="pc-status-field"
            v-model="statusField"
            placeholder="Status"
            :disabled="isBusy"
            data-testid="pc-status-field"
          />
        </div>
        <div class="flex flex-col gap-1 flex-1 min-w-40">
          <label for="pc-epic-type" class="text-sm font-medium">Epic issue type</label>
          <InputText
            id="pc-epic-type"
            v-model="epicIssueType"
            placeholder="Epic"
            :disabled="isBusy"
            data-testid="pc-epic-issue-type"
          />
        </div>
      </div>

      <!-- ── Done options ──────────────────────────────────────────────────── -->
      <div class="flex flex-col gap-1">
        <label for="pc-done-options" class="text-sm font-medium">Done options</label>
        <AutoComplete
          v-model="doneOptions"
          input-id="pc-done-options"
          multiple
          :typeahead="false"
          placeholder='Type a "done" status name and press Enter'
          fluid
          :disabled="isBusy"
          data-testid="pc-done-options"
        />
        <small :style="{ color: 'var(--p-text-muted-color)' }">
          Status option names that mark a story as done on import (case-insensitive).
        </small>
      </div>

      <!-- ── Status mapping ────────────────────────────────────────────────── -->
      <div
        class="flex flex-col gap-3"
        style="
          border: 1px solid var(--surface-border);
          border-radius: 0.375rem;
          padding: 1rem;
        "
      >
        <div class="flex items-center justify-between flex-wrap gap-2">
          <div class="flex flex-col gap-0.5">
            <span class="text-sm font-medium">Status mapping</span>
            <span class="text-xs" :style="{ color: 'var(--p-text-muted-color)' }">
              Map each internal status to the matching GitHub Projects option. Load the live
              options from the board first.
            </span>
          </div>
          <div class="flex items-center gap-2">
            <Button
              label="Auto-fill"
              icon="pi pi-magic"
              severity="secondary"
              text
              size="small"
              :disabled="!optionsLoaded || isBusy"
              data-testid="pc-auto-fill"
              @click="handleAutoFill"
            />
            <Button
              label="Load options"
              icon="pi pi-refresh"
              severity="secondary"
              size="small"
              :loading="fetchStatusOptions.isLoading.value"
              :disabled="isBusy || !projectUrl.trim()"
              data-testid="pc-load-options"
              @click="handleLoadOptions"
            />
          </div>
        </div>

        <Message
          v-if="optionsError"
          severity="error"
          :closable="false"
          data-testid="pc-options-error"
        >
          {{ optionsError }}
        </Message>

        <!-- Mapping selects (only rendered once options are loaded) -->
        <div v-if="optionsLoaded" class="flex flex-col gap-3">
          <div class="flex flex-wrap gap-4">
            <!-- Backlog -->
            <div class="flex flex-col gap-1 flex-1 min-w-36">
              <label for="pc-map-backlog" class="text-xs font-medium">
                <span
                  class="inline-block w-2 h-2 rounded-full mr-1"
                  style="background: var(--p-text-muted-color)"
                />
                Backlog
              </label>
              <Select
                id="pc-map-backlog"
                v-model="mappingBacklog"
                :options="mappingSelectOptions"
                option-label="name"
                option-value="id"
                placeholder="— not mapped —"
                :disabled="isBusy"
                data-testid="pc-map-backlog"
              />
            </div>
            <!-- Running -->
            <div class="flex flex-col gap-1 flex-1 min-w-36">
              <label for="pc-map-running" class="text-xs font-medium">
                <span
                  class="inline-block w-2 h-2 rounded-full mr-1"
                  style="background: var(--status-running-color, var(--p-primary-color))"
                />
                Running
              </label>
              <Select
                id="pc-map-running"
                v-model="mappingRunning"
                :options="mappingSelectOptions"
                option-label="name"
                option-value="id"
                placeholder="— not mapped —"
                :disabled="isBusy"
                data-testid="pc-map-running"
              />
            </div>
          </div>
          <div class="flex flex-wrap gap-4">
            <!-- Done -->
            <div class="flex flex-col gap-1 flex-1 min-w-36">
              <label for="pc-map-done" class="text-xs font-medium">
                <span
                  class="inline-block w-2 h-2 rounded-full mr-1"
                  style="background: var(--status-done-color, #22c55e)"
                />
                Done
              </label>
              <Select
                id="pc-map-done"
                v-model="mappingDone"
                :options="mappingSelectOptions"
                option-label="name"
                option-value="id"
                placeholder="— not mapped —"
                :disabled="isBusy"
                data-testid="pc-map-done"
              />
            </div>
            <!-- Failed -->
            <div class="flex flex-col gap-1 flex-1 min-w-36">
              <label for="pc-map-failed" class="text-xs font-medium">
                <span
                  class="inline-block w-2 h-2 rounded-full mr-1"
                  style="background: var(--status-failed-color, #ef4444)"
                />
                Failed
              </label>
              <Select
                id="pc-map-failed"
                v-model="mappingFailed"
                :options="mappingSelectOptions"
                option-label="name"
                option-value="id"
                placeholder="— not mapped —"
                :disabled="isBusy"
                data-testid="pc-map-failed"
              />
            </div>
          </div>
        </div>

        <!-- Placeholder when options not yet loaded -->
        <p
          v-else
          class="text-sm"
          :style="{ color: 'var(--p-text-muted-color)' }"
          data-testid="pc-options-hint"
        >
          Enter a Project URL and click
          <strong>Load options</strong> to populate the mapping selects.
        </p>
      </div>

      <!-- ── Toggles ───────────────────────────────────────────────────────── -->
      <div class="flex flex-col gap-3">
        <div class="flex items-start gap-3">
          <ToggleSwitch
            v-model="writebackEnabled"
            input-id="pc-writeback-enabled"
            :disabled="isBusy"
            data-testid="pc-writeback-enabled"
          />
          <div class="flex flex-col gap-0.5">
            <label for="pc-writeback-enabled" class="text-sm font-medium">Enable write-back</label>
            <small :style="{ color: 'var(--p-text-muted-color)' }">
              Push internal status transitions to the tracker automatically. Requires a git
              connection and at least one mapped status.
            </small>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <ToggleSwitch
            v-model="postRunComment"
            input-id="pc-post-run-comment"
            :disabled="isBusy"
            data-testid="pc-post-run-comment"
          />
          <div class="flex flex-col gap-0.5">
            <label for="pc-post-run-comment" class="text-sm font-medium">
              Post run comment
            </label>
            <small :style="{ color: 'var(--p-text-muted-color)' }">
              Post a comment linking the hopeitworks run on the tracker item at each status
              transition.
            </small>
          </div>
        </div>
      </div>

      <!-- Error from save -->
      <Message
        v-if="saveError"
        severity="error"
        :closable="false"
        data-testid="pc-save-error"
      >
        {{ saveError }}
      </Message>

      <!-- Actions -->
      <div class="flex justify-end gap-2">
        <Button
          label="Save"
          icon="pi pi-save"
          severity="success"
          :loading="saveConnector.isLoading.value"
          :disabled="isBusy"
          data-testid="pc-save"
          @click="handleSave"
        />
      </div>
    </template>
  </section>
</template>
