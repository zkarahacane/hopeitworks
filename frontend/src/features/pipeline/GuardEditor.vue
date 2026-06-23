<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import type { Guard, GuardKind, GuardOnFail } from '@/stores/pipelineConfig'

const props = defineProps<{
  guards: Guard[]
  isAdmin: boolean
}>()

const emit = defineEmits<{
  add: []
  remove: [index: number]
  update: [index: number, guard: Guard]
}>()

const kindOptions: { label: string; value: GuardKind }[] = [
  { label: 'Log silence', value: 'log_silence' },
  { label: 'Wall clock', value: 'wallclock' },
  { label: 'Cost batch', value: 'cost_batch' },
]

const onFailOptions: { label: string; value: GuardOnFail }[] = [
  { label: 'Halt-gate', value: 'halt-gate' },
  { label: 'Fail', value: 'fail' },
  { label: 'Retry', value: 'retry' },
]

/** Per-kind metadata for the numeric field rendered next to the kind Select. */
const kindMeta: Record<GuardKind, { field: 'threshold' | 'max'; unit: string; hint: string }> = {
  log_silence: { field: 'threshold', unit: 's', hint: 'no agent output for N seconds' },
  wallclock: { field: 'max', unit: 's', hint: 'step running longer than N seconds' },
  cost_batch: { field: 'max', unit: 'USD', hint: 'cumulative run cost over N USD' },
}

const guards = computed(() => props.guards)

function metaFor(kind: GuardKind) {
  return kindMeta[kind]
}

function numericValue(guard: Guard): number | null {
  const field = kindMeta[guard.kind].field
  return guard[field] ?? null
}

function onKindChange(index: number, guard: Guard, kind: GuardKind) {
  if (kind === guard.kind) return
  // Re-home the existing numeric value onto the field the new kind expects so a
  // configured threshold isn't silently lost when switching kinds.
  const current = numericValue(guard) ?? undefined
  const next: Guard = { kind, on_fail: guard.on_fail }
  if (kindMeta[kind].field === 'threshold') {
    next.threshold = current
  } else {
    next.max = current
  }
  emit('update', index, next)
}

function onNumericChange(index: number, guard: Guard, value: number | null) {
  const field = kindMeta[guard.kind].field
  const next: Guard = { ...guard, [field]: value ?? undefined }
  emit('update', index, next)
}

function onFailChange(index: number, guard: Guard, on_fail: GuardOnFail) {
  emit('update', index, { ...guard, on_fail })
}
</script>

<template>
  <div class="flex flex-col gap-2" data-testid="guard-editor">
    <div class="flex items-center justify-between">
      <label class="text-sm font-medium">Guards</label>
      <Button
        v-if="isAdmin"
        label="+ Guard"
        text
        size="small"
        data-testid="add-guard"
        @click="emit('add')"
      />
    </div>

    <p v-if="guards.length === 0" class="text-xs opacity-60" data-testid="guard-empty">
      No guards. Probes park the run on breach (halt-gate) by default.
    </p>

    <div
      v-for="(guard, index) in guards"
      :key="index"
      class="flex flex-wrap items-center gap-2"
      data-testid="guard-row"
    >
      <!-- Kind -->
      <Select
        :model-value="guard.kind"
        :options="kindOptions"
        option-label="label"
        option-value="value"
        :disabled="!isAdmin"
        size="small"
        class="w-36"
        data-testid="guard-kind-select"
        @update:model-value="(v: GuardKind) => onKindChange(index, guard, v)"
      />

      <!-- Kind-dependent numeric field -->
      <div class="flex items-center gap-1">
        <InputNumber
          :model-value="numericValue(guard)"
          :min="0"
          :max-fraction-digits="metaFor(guard.kind).unit === 'USD' ? 2 : 0"
          :disabled="!isAdmin"
          show-buttons
          class="w-28"
          data-testid="guard-numeric-input"
          @update:model-value="(v: number) => onNumericChange(index, guard, v)"
        />
        <span class="text-xs opacity-60" data-testid="guard-unit">{{ metaFor(guard.kind).unit }}</span>
      </div>

      <!-- on_fail -->
      <Select
        :model-value="guard.on_fail"
        :options="onFailOptions"
        option-label="label"
        option-value="value"
        :disabled="!isAdmin"
        size="small"
        class="w-32"
        data-testid="guard-on-fail-select"
        @update:model-value="(v: GuardOnFail) => onFailChange(index, guard, v)"
      />

      <span class="text-xs opacity-50 flex-1 min-w-0" data-testid="guard-hint">
        {{ metaFor(guard.kind).hint }}
      </span>

      <Button
        v-if="isAdmin"
        icon="pi pi-trash"
        text
        rounded
        size="small"
        severity="danger"
        aria-label="Remove guard"
        data-testid="remove-guard"
        @click="emit('remove', index)"
      />
    </div>
  </div>
</template>
