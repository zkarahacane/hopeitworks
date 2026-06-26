<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import Password from 'primevue/password'
import Message from 'primevue/message'
import ConfirmDialog from 'primevue/confirmdialog'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth'
import { formatRelativeDate } from '@/utils/formatDate'
import {
  useGitConnection,
  type GitConnectionStatus,
  type GitConnectionTestResult,
} from './useGitConnection'
import type { Project } from '@/stores/projects'

const props = defineProps<{
  project: Project
}>()

const authStore = useAuthStore()
const confirm = useConfirm()
const toast = useToast()

// Authorization (NORMATIVE §10.0): controls are exposed only to a global admin OR the
// project owner. role comes from the auth store; ownership is project.owner_id === user.id.
// The backend enforces the same rule (403); the UI never exposes the controls otherwise.
const canManage = computed(() => {
  const u = authStore.user
  if (!u) return false
  return u.role === 'admin' || props.project.owner_id === u.id
})

const { status, save, test, clear, statusSeverity } = useGitConnection()

// Last-known advisory status. Always rendered next to last_validated_at — never as a
// bare "connected" (anti-déphasage §10.1).
const conn = ref<GitConnectionStatus | null>(null)
const testResult = ref<GitConnectionTestResult | null>(null)
const token = ref('')

const statusValue = computed(() => conn.value?.status ?? 'unconfigured')

const sourceLabel = computed(() => {
  switch (conn.value?.source) {
    case 'connection':
      return 'via stored token'
    case 'env':
      return 'via environment GITHUB_TOKEN'
    default:
      return 'no token configured'
  }
})

const lastChecked = computed(() =>
  conn.value?.last_validated_at ? formatRelativeDate(conn.value.last_validated_at) : 'never',
)

async function refresh() {
  const result = await status.execute(props.project.id)
  if (result) conn.value = result
}

onMounted(() => {
  if (canManage.value) refresh()
})

async function handleSave() {
  if (!token.value) return
  const result = await save.execute(props.project.id, { token: token.value, validate: true })
  if (result) {
    conn.value = result
    testResult.value = null
    token.value = ''
    toast.add({ severity: 'success', summary: 'Git connection saved', life: 3000 })
  }
}

async function handleTest() {
  const result = await test.execute(props.project.id, token.value || undefined)
  if (result) {
    testResult.value = result
    // Testing the stored token persists status server-side — re-fetch so the card
    // reflects the freshly refreshed last_validated_at.
    await refresh()
  }
}

function handleDisconnect() {
  confirm.require({
    message:
      'Disconnect this project from its git host? Token resolution reverts to the environment fallback (GITHUB_TOKEN).',
    header: 'Disconnect git',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    acceptLabel: 'Disconnect',
    accept: async () => {
      await clear.execute(props.project.id)
      testResult.value = null
      token.value = ''
      await refresh()
      toast.add({ severity: 'success', summary: 'Git connection cleared', life: 3000 })
    },
  })
}

const actionError = computed(
  () => save.error.value?.message || test.error.value?.message || clear.error.value?.message || null,
)
const isBusy = computed(
  () => save.isLoading.value || test.isLoading.value || clear.isLoading.value || status.isLoading.value,
)
</script>

<template>
  <ConfirmDialog />

  <section
    data-testid="git-connection-card"
    class="flex flex-col gap-4"
    style="
      background: var(--surface-raised);
      border: 1px solid var(--surface-border);
      border-radius: 0.5rem;
      padding: 1.5rem;
    "
  >
    <div class="flex items-center justify-between gap-3">
      <div class="flex flex-col gap-1">
        <h3 class="text-sm font-semibold">Git connection</h3>
        <p class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          Store an encrypted Personal Access Token for {{ project.git_provider ?? 'github' }}. The
          token is write-only — only its last 4 characters are ever shown.
        </p>
      </div>
      <Tag
        data-testid="git-connection-status"
        :value="statusValue"
        :severity="statusSeverity(statusValue)"
      />
    </div>

    <!-- Advisory status, always shown next to last_validated_at (anti-déphasage). -->
    <dl class="flex flex-col gap-2 text-sm">
      <div class="flex items-center gap-2">
        <dt :style="{ color: 'var(--p-text-muted-color)' }">Last checked</dt>
        <dd>{{ lastChecked }}</dd>
        <span :style="{ color: 'var(--p-text-muted-color)' }">·</span>
        <dd>{{ sourceLabel }}</dd>
      </div>
      <div v-if="conn?.account_login" class="flex items-center gap-2">
        <dt :style="{ color: 'var(--p-text-muted-color)' }">Account</dt>
        <dd>{{ conn.account_login }}</dd>
      </div>
      <div v-if="conn?.secret_last4" class="flex items-center gap-2">
        <dt :style="{ color: 'var(--p-text-muted-color)' }">Token</dt>
        <dd><code class="text-xs">…{{ conn.secret_last4 }}</code></dd>
        <Tag v-if="conn?.token_type" :value="conn.token_type" severity="secondary" />
      </div>
      <div v-if="conn?.scopes && conn.scopes.length" class="flex items-center gap-2 flex-wrap">
        <dt :style="{ color: 'var(--p-text-muted-color)' }">Scopes</dt>
        <dd class="flex gap-1 flex-wrap">
          <Tag v-for="s in conn.scopes" :key="s" :value="s" severity="secondary" />
        </dd>
      </div>
      <div v-if="conn?.expires_at" class="flex items-center gap-2">
        <dt :style="{ color: 'var(--p-text-muted-color)' }">Expires</dt>
        <dd>{{ formatRelativeDate(conn.expires_at) }}</dd>
      </div>
    </dl>

    <!-- Read-only path for non owner/admin: controls are masked entirely. -->
    <Message
      v-if="!canManage"
      severity="secondary"
      :closable="false"
      data-testid="git-connection-readonly"
    >
      Connecting this project to its git host is managed by the project owner or a global admin.
    </Message>

    <!-- Owner/admin controls. -->
    <template v-else>
      <div class="flex flex-col gap-2">
        <label for="git-connection-token-input" class="text-sm font-medium">
          Personal Access Token
        </label>
        <Password
          input-id="git-connection-token-input"
          v-model="token"
          data-testid="git-connection-token"
          :feedback="false"
          toggle-mask
          fluid
          placeholder="ghp_… or github_pat_…"
          autocomplete="off"
        />
        <small :style="{ color: 'var(--p-text-muted-color)' }">
          Grant <code>read:project</code> (plus <code>repo</code> / <code>read:org</code> for private
          boards). A <strong>fine-grained PAT</strong> is recommended; note that fine-grained PATs
          cannot read user-owned (personal) Projects v2 boards.
        </small>
      </div>

      <Message
        v-if="actionError"
        severity="error"
        :closable="false"
        data-testid="git-connection-error"
      >
        {{ actionError }}
      </Message>

      <!-- Live test result: surface missing_scopes as an advisory warning. -->
      <Message
        v-if="testResult && testResult.missing_scopes && testResult.missing_scopes.length"
        severity="warn"
        :closable="false"
        data-testid="git-connection-missing-scopes"
      >
        Missing scopes: {{ testResult.missing_scopes.join(', ') }}.
        {{ testResult.message }}
      </Message>
      <Message
        v-else-if="testResult"
        :severity="testResult.ok ? 'success' : 'warn'"
        :closable="false"
        data-testid="git-connection-test-result"
      >
        {{ testResult.message || (testResult.ok ? 'Connection OK.' : 'Connection unusable.') }}
        <template v-if="testResult.account_login"> ({{ testResult.account_login }})</template>
      </Message>

      <div class="flex items-center justify-end gap-2 flex-wrap">
        <Button
          label="Disconnect"
          severity="danger"
          text
          icon="pi pi-times"
          data-testid="git-connection-clear"
          :loading="clear.isLoading.value"
          :disabled="isBusy || !conn?.configured"
          @click="handleDisconnect"
        />
        <Button
          label="Test connection"
          severity="secondary"
          icon="pi pi-bolt"
          data-testid="git-connection-test"
          :loading="test.isLoading.value"
          :disabled="isBusy"
          @click="handleTest"
        />
        <Button
          label="Save & verify"
          severity="success"
          icon="pi pi-save"
          data-testid="git-connection-save"
          :loading="save.isLoading.value"
          :disabled="isBusy || !token"
          @click="handleSave"
        />
      </div>
    </template>
  </section>
</template>
