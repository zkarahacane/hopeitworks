<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Message from 'primevue/message'
import SelectButton from 'primevue/selectbutton'
import InputText from 'primevue/inputtext'
import AutoComplete from 'primevue/autocomplete'
import { usePlanningImport, type PlanningSource } from '@/composables/usePlanningImport'
import { useGitConnection, type GitConnectionStatus } from '@/features/projects/useGitConnection'
import { usePlanningConnector } from '@/composables/usePlanningConnector'

const props = defineProps<{
  visible: boolean
  projectId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  imported: []
}>()

// ── Import-flow guard (GitHub Projects) ──────────────────────────────────────
// GitHub imports need a live connection. Fetch the advisory status when the dialog
// opens; if it isn't `connected`, gate the source (canSubmit stays false) and surface
// an inline "connect first" link to settings instead of an opaque 422 on import.
const { status: gitStatus } = useGitConnection()
const gitConn = ref<GitConnectionStatus | null>(null)
const githubConnected = computed(() => gitConn.value?.status === 'connected')

// ── Persisted connector pre-fill (GitHub Projects) ───────────────────────────
// When a PlanningConnector is saved for this project, pre-fill the GitHub form
// fields so the user doesn't have to re-enter them on every import.
const { fetchConnector } = usePlanningConnector()

async function refreshGitConnection() {
  const [gitResult, connResult] = await Promise.all([
    gitStatus.execute(props.projectId),
    fetchConnector.execute(props.projectId),
  ])
  gitConn.value = gitResult
  // Pre-fill GitHub fields from the persisted connector (non-destructive: only
  // fills when the field is currently empty so a user mid-edit isn't overwritten).
  if (connResult) {
    if (!projectUrl.value && connResult.project_url) {
      projectUrl.value = connResult.project_url
    }
    if (!statusField.value || statusField.value === 'Status') {
      statusField.value = connResult.status_field ?? 'Status'
    }
    if (!doneOptions.value.length && connResult.done_options?.length) {
      doneOptions.value = [...connResult.done_options]
    }
    if (!epicIssueType.value || epicIssueType.value === 'Epic') {
      epicIssueType.value = connResult.epic_issue_type ?? 'Epic'
    }
  }
}

watch(
  () => props.visible,
  (open) => {
    if (open) refreshGitConnection()
  },
  { immediate: true },
)

const {
  source,
  canSubmit,
  fileContent,
  fileName,
  parsedPreview,
  fileError,
  selectFile,
  projectUrl,
  statusField,
  doneOptions,
  epicIssueType,
  result,
  committed,
  apiError,
  isLoading,
  preview,
  commit,
  reset,
} = usePlanningImport({ githubConnected })

const sourceOptions: { label: string; value: PlanningSource }[] = [
  { label: 'Markdown', value: 'markdown' },
  { label: 'GitHub Projects', value: 'github_projects' },
]

const isDragging = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const validStories = computed(() => parsedPreview.value.filter((s) => s.valid))
const invalidStories = computed(() => parsedPreview.value.filter((s) => !s.valid))
const validCount = computed(() => validStories.value.length)
const invalidCount = computed(() => invalidStories.value.length)

/** PrimeVue Tag severity for each per-node import action. */
const actionSeverity: Record<string, 'success' | 'info' | 'warn' | 'secondary' | 'danger'> = {
  create: 'success',
  update: 'info',
  lock: 'warn',
  skip: 'secondary',
  fail: 'danger',
}

/** A compact tally summarising the result counts (created / updated / unchanged / locked / failed). */
const tally = computed(() => {
  const r = result.value
  if (!r) return null
  return {
    created: r.epics_created + r.stories_created,
    updated: r.epics_updated + r.stories_updated,
    skipped: r.skipped,
    locked: r.locked,
    failed: r.failed,
  }
})

function triggerFileInput() {
  fileInputRef.value?.click()
}

function handleDrop(event: DragEvent) {
  isDragging.value = false
  const file = event.dataTransfer?.files[0]
  if (file) {
    selectFile(file)
  }
}

function handleFileInput(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (file) {
    selectFile(file)
  }
  target.value = ''
}

async function handlePreview() {
  await preview(props.projectId)
}

async function handleImport() {
  const r = await commit(props.projectId)
  if (r) {
    // Tell the host to refresh the board immediately; the result panel stays open.
    emit('imported')
  }
}

function handleImportAnother() {
  reset()
}

function handleClose() {
  reset()
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Import planning"
    class="w-full max-w-3xl"
    @update:visible="handleClose"
  >
    <div class="flex flex-col gap-4">
      <!-- ── Source picker ───────────────────────────────────────────────────── -->
      <div class="flex flex-col gap-1">
        <span style="font-size: 0.78rem; color: var(--p-text-muted-color)">Source</span>
        <SelectButton
          v-model="source"
          :options="sourceOptions"
          option-label="label"
          option-value="value"
          :allow-empty="false"
          aria-label="Import source"
          data-testid="source-picker"
        />
      </div>

      <!-- ── Markdown source ─────────────────────────────────────────────────── -->
      <div v-if="source === 'markdown'" class="flex flex-col gap-3" data-testid="markdown-panel">
        <div
          class="flex flex-col items-center justify-center gap-3 p-8 border-2 border-dashed rounded-lg cursor-pointer"
          :style="{
            borderColor: isDragging ? 'var(--p-primary-color)' : 'var(--surface-border)',
            backgroundColor: isDragging ? 'var(--p-primary-50)' : 'transparent',
          }"
          data-testid="drop-zone"
          @click="triggerFileInput"
          @dragover.prevent
          @dragenter="isDragging = true"
          @dragleave="isDragging = false"
          @drop.prevent="handleDrop"
        >
          <i class="pi pi-upload" style="font-size: 2rem; color: var(--p-text-muted-color)" />
          <p style="color: var(--p-text-muted-color)">
            Drag & drop a .md file here, or click to browse
          </p>
          <p v-if="fileName" class="text-sm font-medium">{{ fileName }}</p>
        </div>
        <input
          ref="fileInputRef"
          type="file"
          accept=".md"
          class="hidden"
          data-testid="file-input"
          @change="handleFileInput"
        />
        <small
          v-if="fileError"
          :style="{ color: 'var(--status-failed-color)' }"
          data-testid="file-error"
        >{{ fileError }}</small>

        <!-- Local (client-side) parse preview -->
        <div v-if="fileContent" class="flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <Tag :value="`${validCount} stories detected`" severity="info" />
            <Tag v-if="invalidCount > 0" :value="`${invalidCount} invalid`" severity="warn" />
          </div>
          <DataTable
            v-if="validStories.length > 0"
            :value="validStories"
            size="small"
            data-testid="preview-table"
          >
            <Column field="key" header="Key" />
            <Column field="title" header="Title" />
            <Column field="scope" header="Scope" />
          </DataTable>
          <div v-if="invalidStories.length > 0" data-testid="invalid-stories">
            <h4 class="text-sm font-semibold mb-2" style="color: var(--p-text-muted-color)">
              Invalid stories
            </h4>
            <ul class="list-disc pl-5 text-sm">
              <li
                v-for="(story, idx) in invalidStories"
                :key="idx"
                :style="{ color: 'var(--status-failed-color)' }"
              >
                {{ story.key }}: {{ story.error }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- ── GitHub Projects source ──────────────────────────────────────────── -->
      <div v-else class="flex flex-col gap-3" data-testid="github-panel">
        <!-- Import-flow guard: block until the project is connected to GitHub. -->
        <Message
          v-if="!githubConnected"
          severity="warn"
          :closable="false"
          data-testid="github-connection-guard"
        >
          This project is not connected to GitHub.
          <RouterLink
            :to="{ name: 'project-settings', params: { id: projectId } }"
            data-testid="github-connection-guard-link"
            @click="handleClose"
          >
            Connect this project to GitHub first →
          </RouterLink>
        </Message>
        <div class="flex flex-col gap-1">
          <label for="gh-project-url" style="font-size: 0.78rem; color: var(--p-text-muted-color)">
            Project URL
          </label>
          <InputText
            id="gh-project-url"
            v-model="projectUrl"
            placeholder="https://github.com/orgs/<org>/projects/<n>"
            class="w-full"
            data-testid="github-project-url"
          />
        </div>
        <div class="flex flex-wrap gap-3">
          <div class="flex flex-col gap-1">
            <label for="gh-status-field" style="font-size: 0.78rem; color: var(--p-text-muted-color)">
              Status field
            </label>
            <InputText
              id="gh-status-field"
              v-model="statusField"
              placeholder="Status"
              data-testid="github-status-field"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label for="gh-epic-type" style="font-size: 0.78rem; color: var(--p-text-muted-color)">
              Epic issue type
            </label>
            <InputText
              id="gh-epic-type"
              v-model="epicIssueType"
              placeholder="Epic"
              data-testid="github-epic-issue-type"
            />
          </div>
        </div>
        <div class="flex flex-col gap-1">
          <label for="gh-done-options" style="font-size: 0.78rem; color: var(--p-text-muted-color)">
            Done options
          </label>
          <AutoComplete
            v-model="doneOptions"
            input-id="gh-done-options"
            multiple
            :typeahead="false"
            placeholder="Type a status option (e.g. Done) and press Enter"
            class="w-full"
            data-testid="github-done-options"
          />
          <small style="color: var(--p-text-muted-color)">
            Status option names that mean "done". Leave empty to map everything to backlog.
          </small>
        </div>
      </div>

      <!-- ── Import decisions (dry-run preview or committed result) ─────────────── -->
      <div v-if="result" class="flex flex-col gap-2" data-testid="import-result">
        <div class="flex items-center gap-2 flex-wrap" data-testid="preview-tally">
          <span style="font-weight: 600; font-size: 0.85rem">
            {{ committed ? 'Imported' : 'Preview' }}
          </span>
          <Tag v-if="tally" :value="`${tally.created} created`" severity="success" />
          <Tag v-if="tally" :value="`${tally.updated} updated`" severity="info" />
          <Tag v-if="tally && tally.skipped > 0" :value="`${tally.skipped} unchanged`" severity="secondary" />
          <Tag v-if="tally && tally.locked > 0" :value="`${tally.locked} locked`" severity="warn" />
          <Tag v-if="tally && tally.failed > 0" :value="`${tally.failed} failed`" severity="danger" />
        </div>

        <DataTable
          :value="result.items"
          size="small"
          data-testid="preview-result-table"
        >
          <Column field="key" header="Key" />
          <Column field="kind" header="Kind" />
          <Column header="Action">
            <template #body="{ data }">
              <Tag :value="data.action" :severity="actionSeverity[data.action] ?? 'secondary'" />
            </template>
          </Column>
          <Column field="mapped_status" header="Status" />
          <Column header="Source">
            <template #body="{ data }">
              <a
                v-if="data.source_url"
                :href="data.source_url"
                target="_blank"
                rel="noopener"
                data-testid="item-source-link"
              >
                <i class="pi pi-external-link" style="font-size: 0.75rem" aria-hidden="true" /> open
              </a>
              <span v-else style="color: var(--p-text-muted-color)">—</span>
            </template>
          </Column>
          <Column field="reason" header="Reason" />
        </DataTable>
      </div>

      <Message v-if="apiError" severity="error" :closable="false" data-testid="api-error">
        {{ apiError }}
      </Message>
    </div>

    <template #footer>
      <!-- Committed: only Close + Import another -->
      <div v-if="committed" class="flex justify-end gap-2">
        <Button
          label="Import another"
          severity="secondary"
          text
          data-testid="import-another-button"
          @click="handleImportAnother"
        />
        <Button label="Close" data-testid="close-button" @click="handleClose" />
      </div>

      <!-- Configuring / previewing -->
      <div v-else class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text data-testid="cancel-button" @click="handleClose" />
        <Button
          label="Preview"
          icon="pi pi-eye"
          severity="secondary"
          :loading="isLoading"
          :disabled="!canSubmit"
          data-testid="preview-button"
          @click="handlePreview"
        />
        <Button
          label="Import"
          icon="pi pi-upload"
          :loading="isLoading"
          :disabled="!canSubmit"
          data-testid="import-button"
          @click="handleImport"
        />
      </div>
    </template>
  </Dialog>
</template>
