<script setup lang="ts">
import { computed } from 'vue'
import * as Diff2Html from 'diff2html'
import 'diff2html/bundles/css/diff2html.min.css'
import Button from 'primevue/button'
import Message from 'primevue/message'

const props = defineProps<{
  diff: string | null | undefined
  mode: 'side-by-side' | 'line-by-line'
}>()

const emit = defineEmits<{
  'update:mode': [mode: 'side-by-side' | 'line-by-line']
}>()

const html = computed(() => {
  if (!props.diff) return null
  return Diff2Html.html(props.diff, {
    drawFileList: true,
    matching: 'lines',
    outputFormat: props.mode,
  })
})

function toggleMode() {
  emit('update:mode', props.mode === 'side-by-side' ? 'line-by-line' : 'side-by-side')
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div v-if="html" class="flex flex-col gap-2">
      <div class="flex justify-end">
        <Button
          :label="mode === 'side-by-side' ? 'Unified' : 'Side by side'"
          size="small"
          severity="secondary"
          @click="toggleMode"
        />
      </div>
      <div v-html="html" />
    </div>
    <Message v-else severity="info" :closable="false">
      No diff available
    </Message>
  </div>
</template>
