<script setup lang="ts">
import { ref, computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Message from 'primevue/message'
import { useStoryImport } from '@/composables/useStoryImport'

const props = defineProps<{
  visible: boolean
  projectId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  imported: []
}>()

const {
  fileContent,
  fileName,
  parsedPreview,
  importResult,
  fileError,
  apiError,
  isImporting,
  selectFile,
  importStories,
  reset,
} = useStoryImport()

const isDragging = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

const validStories = computed(() => parsedPreview.value.filter((s) => s.valid))
const invalidStories = computed(() => parsedPreview.value.filter((s) => !s.valid))
const validCount = computed(() => validStories.value.length)
const invalidCount = computed(() => invalidStories.value.length)

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

async function handleImport() {
  await importStories(props.projectId)
}

function handleImportAnother() {
  reset()
}

function handleClose() {
  if (importResult.value) {
    emit('imported')
  }
  reset()
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Import Stories"
    class="w-full max-w-2xl"
    @update:visible="handleClose"
  >
    <div class="flex flex-col gap-4">
      <!-- Step 1: Upload zone -->
      <div v-if="!fileContent && !importResult">
        <div
          class="flex flex-col items-center justify-center gap-3 p-8 border-2 border-dashed rounded-lg cursor-pointer"
          :style="{
            borderColor: isDragging ? 'var(--p-primary-color)' : 'var(--p-surface-300)',
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
        <small v-if="fileError" class="text-red-500" data-testid="file-error">{{
          fileError
        }}</small>
      </div>

      <!-- Step 2: Preview -->
      <div v-if="fileContent && !importResult">
        <div class="flex items-center gap-2 mb-4">
          <Tag :value="`${validCount} stories detected`" severity="info" />
          <Tag
            v-if="invalidCount > 0"
            :value="`${invalidCount} invalid`"
            severity="warn"
          />
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

        <div v-if="invalidStories.length > 0" class="mt-4" data-testid="invalid-stories">
          <h4 class="text-sm font-semibold mb-2" style="color: var(--p-text-muted-color)">
            Invalid stories
          </h4>
          <ul class="list-disc pl-5 text-sm">
            <li v-for="(story, idx) in invalidStories" :key="idx" class="text-red-500">
              {{ story.key }}: {{ story.error }}
            </li>
          </ul>
        </div>

        <Message v-if="apiError" severity="error" :closable="false" class="mt-4" data-testid="api-error">
          {{ apiError }}
        </Message>
      </div>

      <!-- Step 3: Result -->
      <div v-if="importResult" data-testid="import-result">
        <div class="flex items-center gap-2 mb-4">
          <Tag :value="`${importResult.imported} created`" severity="success" />
          <Tag :value="`${importResult.updated} updated`" severity="info" />
          <Tag
            v-if="importResult.failed > 0"
            :value="`${importResult.failed} failed`"
            severity="danger"
          />
        </div>

        <div v-if="importResult.errors.length > 0" class="mt-2" data-testid="import-errors">
          <h4 class="text-sm font-semibold mb-2" style="color: var(--p-text-muted-color)">
            Errors
          </h4>
          <ul class="list-disc pl-5 text-sm">
            <li v-for="(err, idx) in importResult.errors" :key="idx" class="text-red-500">
              {{ err.key }}: {{ err.message }}
            </li>
          </ul>
        </div>
      </div>
    </div>

    <template #footer>
      <!-- Step 1 footer: no buttons needed (upload zone is interactive) -->

      <!-- Step 2 footer: Cancel + Import -->
      <div v-if="fileContent && !importResult" class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="handleClose" />
        <Button
          label="Import"
          icon="pi pi-upload"
          :loading="isImporting"
          :disabled="validCount === 0"
          data-testid="import-button"
          @click="handleImport"
        />
      </div>

      <!-- Step 3 footer: Import Another + Close -->
      <div v-if="importResult" class="flex justify-end gap-2">
        <Button
          label="Import Another File"
          severity="secondary"
          text
          data-testid="import-another-button"
          @click="handleImportAnother"
        />
        <Button label="Close" data-testid="close-button" @click="handleClose" />
      </div>
    </template>
  </Dialog>
</template>
