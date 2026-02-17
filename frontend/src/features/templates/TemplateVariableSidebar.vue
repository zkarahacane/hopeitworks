<script setup lang="ts">
interface EditorRef {
  insertAtCursor: (text: string) => void
}

interface Props {
  editorRef: EditorRef | null
}

const props = defineProps<Props>()

const variables = [
  { name: 'story_key', description: 'Unique story identifier (e.g., S-14)' },
  { name: 'story_title', description: 'Story title/summary' },
  { name: 'story_objective', description: 'Story objective text' },
  { name: 'target_files', description: 'Array of target file paths' },
  { name: 'acceptance_criteria', description: 'Story acceptance criteria text' },
  { name: 'error_context', description: 'Error output from previous failed run (retry only)' },
  { name: 'diff_content', description: 'Git diff from previous attempt (retry/review only)' },
  { name: 'branch_name', description: 'Git branch name for this run' },
  { name: 'repo_url', description: 'Git repository URL' },
]

function formatPlaceholder(name: string): string {
  return `{{${name}}}`
}

function handleVariableClick(variableName: string) {
  if (!props.editorRef) return
  props.editorRef.insertAtCursor(`{{${variableName}}}`)
}
</script>

<template>
  <div class="flex flex-col gap-1 p-3">
    <h3 class="mb-2 text-sm font-semibold text-surface-500">Context Variables</h3>
    <button
      v-for="variable in variables"
      :key="variable.name"
      class="cursor-pointer rounded-md p-2 text-left transition-colors hover:bg-surface-100 dark:hover:bg-surface-700"
      :title="variable.description"
      @click="handleVariableClick(variable.name)"
    >
      <div class="text-sm font-mono font-medium text-primary-600 dark:text-primary-400">
        {{ formatPlaceholder(variable.name) }}
      </div>
      <div class="text-xs text-surface-500">{{ variable.description }}</div>
    </button>
  </div>
</template>
