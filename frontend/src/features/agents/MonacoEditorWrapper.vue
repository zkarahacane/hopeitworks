<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, shallowRef, computed } from 'vue'
import * as monaco from 'monaco-editor'
import { useTheme } from '@/composables/useTheme'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

/** Configure Monaco web workers for Vite bundling */
self.MonacoEnvironment = {
  getWorker(_: unknown, label: string) {
    if (label === 'json') return new jsonWorker()
    if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
    if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
    if (label === 'typescript' || label === 'javascript') return new tsWorker()
    return new editorWorker()
  },
}

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

const container = ref<HTMLDivElement | null>(null)
const editorInstance = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
const { resolvedScheme } = useTheme()

const editorTheme = computed(() => (resolvedScheme.value === 'dark' ? 'vs-dark' : 'vs'))

/**
 * Guard flag to prevent emit loops.
 * When we programmatically set model content (e.g. from prop sync),
 * we set this flag so the onDidChangeModelContent handler skips the emit.
 */
let suppressChangeEvent = false

onMounted(() => {
  if (!container.value) return

  const editor = monaco.editor.create(container.value, {
    value: props.modelValue ?? '',
    language: props.language,
    theme: editorTheme.value,
    readOnly: props.readonly,
    minimap: { enabled: false },
    wordWrap: 'on',
    automaticLayout: true,
    scrollBeyondLastLine: false,
    fontSize: 14,
    lineNumbers: 'on',
    renderWhitespace: 'boundary',
    tabSize: 2,
  })

  editorInstance.value = editor

  editor.onDidChangeModelContent(() => {
    if (suppressChangeEvent) return
    const value = editor.getValue()
    if (value !== props.modelValue) {
      emit('update:modelValue', value)
    }
  })
})

onBeforeUnmount(() => {
  editorInstance.value?.dispose()
  editorInstance.value = null
})

/** Sync external prop changes into the editor (e.g. template switch) */
watch(
  () => props.modelValue,
  (newVal) => {
    const editor = editorInstance.value
    if (!editor) return
    const currentValue = editor.getValue()
    if (newVal !== currentValue) {
      suppressChangeEvent = true
      editor.setValue(newVal ?? '')
      suppressChangeEvent = false
    }
  },
)

watch(
  () => props.readonly,
  (readOnly) => {
    editorInstance.value?.updateOptions({ readOnly })
  },
)

watch(
  () => props.language,
  (language) => {
    const model = editorInstance.value?.getModel()
    if (model && language) {
      monaco.editor.setModelLanguage(model, language)
    }
  },
)

/** React to theme changes: update Monaco editor theme */
watch(
  () => editorTheme.value,
  (theme) => {
    const editor = editorInstance.value
    if (editor) {
      editor.updateOptions({ theme })
    }
  },
)

/** Insert text at the current cursor position without triggering infinite loops */
function insertAtCursor(text: string) {
  const editor = editorInstance.value
  if (!editor) return

  editor.focus()

  const selection = editor.getSelection()
  if (!selection) return

  const op: monaco.editor.IIdentifiedSingleEditOperation = {
    range: selection,
    text,
    forceMoveMarkers: true,
  }

  editor.executeEdits('insertAtCursor', [op])

  // Move cursor to end of inserted text
  const endPosition = editor.getModel()?.getPositionAt(
    editor.getModel()!.getOffsetAt(selection.getStartPosition()) + text.length,
  )
  if (endPosition) {
    editor.setPosition(endPosition)
  }
}

defineExpose({ insertAtCursor })
</script>

<template>
  <div ref="container" class="h-full w-full" />
</template>
