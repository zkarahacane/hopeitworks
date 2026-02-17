<script setup lang="ts">
import { ref, watch } from 'vue'
import { Editor } from '@guolao/vue-monaco-editor'
import type * as monacoEditor from 'monaco-editor'

interface Props {
  modelValue: string
  readonly?: boolean
  language?: string
}

const props = withDefaults(defineProps<Props>(), {
  readonly: false,
  language: 'handlebars',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const editorInstance = ref<monacoEditor.editor.IStandaloneCodeEditor | null>(null)
const monacoInstance = ref<typeof monacoEditor | null>(null)

const editorOptions = {
  readOnly: props.readonly,
  minimap: { enabled: false },
  wordWrap: 'on' as const,
  automaticLayout: true,
  scrollBeyondLastLine: false,
  fontSize: 14,
  lineNumbers: 'on' as const,
  renderWhitespace: 'boundary' as const,
  tabSize: 2,
}

/** Update readonly option when the prop changes */
watch(
  () => props.readonly,
  (readOnly) => {
    editorInstance.value?.updateOptions({ readOnly })
  },
)

function handleMount(editor: monacoEditor.editor.IStandaloneCodeEditor, monaco: typeof monacoEditor) {
  editorInstance.value = editor
  monacoInstance.value = monaco
}

function handleChange(value: string | undefined) {
  emit('update:modelValue', value ?? '')
}

/** Insert text at the current cursor position */
function insertAtCursor(text: string) {
  const editor = editorInstance.value
  const monaco = monacoInstance.value
  if (!editor || !monaco) return

  const selection = editor.getSelection()
  if (!selection) return

  const range = new monaco.Range(
    selection.startLineNumber,
    selection.startColumn,
    selection.endLineNumber,
    selection.endColumn,
  )

  editor.executeEdits('', [{ range, text, forceMoveMarkers: true }])
  editor.focus()
}

defineExpose({ insertAtCursor })
</script>

<template>
  <Editor
    :value="modelValue"
    :language="language"
    theme="vs-dark"
    :options="editorOptions"
    height="100%"
    @mount="handleMount"
    @change="handleChange"
  />
</template>
