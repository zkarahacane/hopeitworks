import { ref, computed, onMounted } from 'vue'
import { useEnvironmentsStore } from '@/stores/environments'
import type { EnvironmentInput, EnvironmentSource } from '@/api/environment'

/** A key/value pair used for editing env vars and commands in the form. */
export interface KVPair {
  key: string
  value: string
}

/** An editable service entry with env vars as a KV pair list. */
export interface EditableService {
  name: string
  image: string
  envPairs: KVPair[]
}

/** Convert a Record<string,string> to an array of KV pairs. */
function recordToKVPairs(record: Record<string, string>): KVPair[] {
  return Object.entries(record).map(([key, value]) => ({ key, value }))
}

/** Convert an array of KV pairs to a Record<string,string>, dropping empty keys. */
function kvPairsToRecord(pairs: KVPair[]): Record<string, string> {
  const result: Record<string, string> = {}
  for (const { key, value } of pairs) {
    if (key.trim() !== '') {
      result[key.trim()] = value
    }
  }
  return result
}

/**
 * Composable for the environment editor form.
 * Handles loading, form state, validation, save, and delete.
 */
export function useEnvironmentEditor(projectId: string) {
  const store = useEnvironmentsStore()

  // Form state
  const stacks = ref<string[]>([])
  const source = ref<EnvironmentSource>('declared')
  const services = ref<EditableService[]>([])
  const commandsPairs = ref<KVPair[]>([])

  const isLoading = computed(() => store.isLoading)
  const isSaving = computed(() => store.isSaving)
  const error = computed(() => store.error)

  /** Whether an environment already exists (has an id). */
  const exists = computed(() => store.environment !== null && store.environment.id !== '')

  /** Validation: every service must have a non-empty image. */
  const canSave = computed(() =>
    services.value.every((s) => s.image.trim() !== ''),
  )

  /** Seed form state from the fetched environment or empty defaults. */
  function seedForm() {
    const env = store.environment
    if (env) {
      stacks.value = [...env.stacks]
      source.value = (env.source as EnvironmentSource) ?? 'declared'
      services.value = env.services.map((s) => ({
        name: s.name,
        image: s.image,
        envPairs: recordToKVPairs(s.env),
      }))
      commandsPairs.value = recordToKVPairs(env.commands)
    } else {
      stacks.value = []
      source.value = 'declared'
      services.value = []
      commandsPairs.value = []
    }
  }

  /** Build an EnvironmentInput from the current form state. */
  function buildInput(): EnvironmentInput {
    return {
      stacks: stacks.value,
      source: source.value,
      services: services.value
        .filter((s) => s.image.trim() !== '')
        .map((s) => ({
          name: s.name,
          image: s.image.trim(),
          env: kvPairsToRecord(s.envPairs),
        })),
      commands: kvPairsToRecord(commandsPairs.value),
    }
  }

  // Service helpers
  function addService() {
    services.value.push({ name: '', image: '', envPairs: [] })
  }

  function removeService(index: number) {
    services.value.splice(index, 1)
  }

  // Per-service env var helpers
  function addEnvPair(serviceIndex: number) {
    const svc = services.value[serviceIndex]
    if (svc) svc.envPairs.push({ key: '', value: '' })
  }

  function removeEnvPair(serviceIndex: number, pairIndex: number) {
    const svc = services.value[serviceIndex]
    if (svc) svc.envPairs.splice(pairIndex, 1)
  }

  // Commands helpers
  function addCommand() {
    commandsPairs.value.push({ key: '', value: '' })
  }

  function removeCommand(index: number) {
    commandsPairs.value.splice(index, 1)
  }

  /** Save the environment. Returns the saved environment or null on failure. */
  async function save() {
    if (!canSave.value) return null
    const input = buildInput()
    return store.saveEnvironment(projectId, input)
  }

  /** Delete the environment. Returns true on success. */
  async function remove() {
    return store.removeEnvironment(projectId)
  }

  onMounted(async () => {
    await store.fetchEnvironment(projectId)
    seedForm()
  })

  return {
    // Form state
    stacks,
    source,
    services,
    commandsPairs,
    // Status
    isLoading,
    isSaving,
    error,
    exists,
    canSave,
    // Helpers
    addService,
    removeService,
    addEnvPair,
    removeEnvPair,
    addCommand,
    removeCommand,
    // Actions
    save,
    remove,
    // Exposed for testing
    buildInput,
    seedForm,
  }
}
