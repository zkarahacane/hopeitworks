# Story 9.5: [FRONT] Notification Settings Page

Status: ready-for-dev

## Story

As a project admin, I want to configure notification channels for my project from the UI, So that the team receives Discord or webhook alerts for key pipeline events without manual API calls.

## Acceptance Criteria (BDD)

**AC1: Notification settings route exists under project settings**
- **Given** I navigate to `/projects/:id/settings/notifications`
- **When** the page loads
- **Then** the notification settings page renders with a list of configured channels
- **And** a "Notifications" link or tab is visible within the project settings area

**AC2: Channel list displays existing configs**
- **Given** the project has notification configs in the database
- **When** the page renders
- **Then** each config shows: channel type badge (Discord/Webhook), masked URL (last 6 chars visible), enabled/disabled toggle, subscribed events as chips, Edit and Delete action buttons
- **And** admin users see Edit/Delete/Add buttons; non-admin users see a read-only list with no action buttons

**AC3: Add Channel dialog creates a new config**
- **Given** I am a project admin on the notifications page
- **When** I click "Add Channel" and fill in the form with valid data
- **Then** `POST /api/v1/projects/{projectId}/notifications` is called with `channel_type`, `config.url`, `events_filter`, `enabled: true`
- **And** on success, the dialog closes and the new channel appears in the list
- **And** a Toast "Channel added" is shown

**AC4: Form validation on Add Channel**
- **Given** the Add Channel dialog is open
- **When** I submit with URL not starting with `https://`
- **Then** an inline error "URL must start with https://" appears below the URL field
- **And** the form is NOT submitted

**AC5: Enable/disable toggle updates via PUT**
- **Given** a channel is displayed in the list
- **When** I toggle the enabled switch
- **Then** `PUT /api/v1/projects/{projectId}/notifications/{id}` is called with `enabled: !current`
- **And** the toggle updates optimistically and reverts on API error with an error Toast

**AC6: Delete with confirmation**
- **Given** I click Delete on a channel
- **When** the PrimeVue ConfirmDialog appears and I confirm
- **Then** `DELETE /api/v1/projects/{projectId}/notifications/{id}` is called
- **And** on success, the channel is removed from the list and a Toast "Channel deleted" is shown
- **And** cancelling the dialog leaves the list unchanged

**AC7: Test notification sends a test event**
- **Given** I click "Test" on a channel
- **When** the request completes
- **Then** `POST /api/v1/projects/{projectId}/notifications/{id}/test` is called
- **And** a Toast "Test sent" (success) or "Test failed: {reason}" (error) is shown

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add notification settings route + navigation (AC: #1)
  - [ ] In `frontend/src/router/index.ts`: add child route under project detail: `{ path: 'settings/notifications', name: 'project-notifications', component: () => import('@/views/NotificationSettingsView.vue') }`
  - [ ] Add "Notifications" entry to the project settings navigation (in `ProjectDetailView.vue` or a dedicated settings layout component)

- [ ] [FRONT] Task 2: Extend openapi.yaml with test endpoint (AC: #7)
  - [ ] Add `POST /projects/{projectId}/notifications/{id}/test` to `api/openapi.yaml` (returns 204 No Content)
  - [ ] Run `cd frontend && npm run generate-api` to update types

- [ ] [FRONT] Task 3: Create `useNotifications` composable (AC: #2, #3, #5, #6, #7)
  - [ ] File: `frontend/src/composables/useNotifications.ts`
  - [ ] Accept `projectId: string`
  - [ ] Expose: `configs`, `isLoading`, `error`, `fetchConfigs()`, `createConfig(payload)`, `updateConfig(id, payload)`, `deleteConfig(id)`, `testConfig(id)`
  - [ ] `toggleEnabled(config)`: calls `updateConfig` with `enabled: !config.enabled`, optimistic update
  - [ ] Unit test: `frontend/src/composables/__tests__/useNotifications.spec.ts`

- [ ] [FRONT] Task 4: Create `AddChannelDialog.vue` component (AC: #3, #4)
  - [ ] File: `frontend/src/features/notifications/AddChannelDialog.vue`
  - [ ] Props: `visible: boolean`, `projectId: string`
  - [ ] Emits: `update:visible`, `created: [config: NotificationConfig]`
  - [ ] Form fields: channel type `Select` (options: Discord, Webhook), URL `InputText`, events multi-select `MultiSelect` (options: `run.completed`, `run.failed`, `hitl_gate.pending`, `circuit_breaker.triggered`), enabled toggle `ToggleSwitch` (default: true)
  - [ ] Validation with vee-validate + zod: URL required + must start with `https://`
  - [ ] On valid submit: call `useNotifications().createConfig(payload)`, emit `created` on success
  - [ ] API error: `<Message severity="error">` inside dialog

- [ ] [FRONT] Task 5: Create `NotificationChannelRow.vue` component (AC: #2, #5, #6, #7)
  - [ ] File: `frontend/src/features/notifications/NotificationChannelRow.vue`
  - [ ] Props: `config: NotificationConfig`, `isAdmin: boolean`
  - [ ] Displays: `Tag` for channel type, masked URL (`****` + last 6 chars), events as `Chip` list, `ToggleSwitch` bound to `config.enabled`
  - [ ] Emits: `toggle`, `edit`, `delete`, `test`
  - [ ] Action buttons (Edit, Delete, Test) only rendered when `isAdmin === true`

- [ ] [FRONT] Task 6: Create `NotificationSettingsView.vue` (AC: #1, #2, #3, #6, #7)
  - [ ] File: `frontend/src/views/NotificationSettingsView.vue`
  - [ ] Injects `projectId` from route params and `project` from `inject('project')`
  - [ ] `isAdmin` derived from auth store user role
  - [ ] Composes: page header with "Add Channel" Button (admin only), list of `NotificationChannelRow`, `AddChannelDialog`
  - [ ] On `@toggle`: call `toggleEnabled(config)`, revert + Toast on error
  - [ ] On `@delete`: show `useConfirm()` dialog, call `deleteConfig(id)` on confirm, Toast on success
  - [ ] On `@test`: call `testConfig(id)`, Toast "Test sent" / "Test failed: {reason}"
  - [ ] Loading: `Skeleton` list while `isLoading`
  - [ ] Empty state: `EmptyState` with message "No notification channels configured" when list is empty

- [ ] [FRONT] Task 7: Mask URL utility function (AC: #2)
  - [ ] Create `frontend/src/utils/maskUrl.ts`: `export function maskUrl(url: string): string` — returns `****${url.slice(-6)}` or full URL if shorter than 6 chars

- [ ] [FRONT] Task 8: Unit tests for useNotifications composable (AC: #3, #5, #7)
  - [ ] `frontend/src/composables/__tests__/useNotifications.spec.ts`
  - [ ] `fetchConfigs` populates `configs.value` from API response
  - [ ] `createConfig` pushes to `configs.value` on success
  - [ ] `toggleEnabled` updates optimistically, reverts on API error
  - [ ] `deleteConfig` removes item from `configs.value` on success
  - [ ] `testConfig` returns resolved promise on success, rejects on error

- [ ] [FRONT] Task 9: Zod schema + form validation tests (AC: #4)
  - [ ] Test zod schema for Add Channel form:
    - Valid: `{ channel_type: 'discord', url: 'https://discord.com/api/webhooks/123', events_filter: ['run.completed'] }`
    - Invalid: URL starting with `http://` → fails validation
    - Invalid: empty URL → fails validation
    - Invalid: empty events_filter → still valid (no events = disabled notifications)

- [ ] [FRONT] Task 10: Lint + type check validation
  - [ ] Run `cd frontend && npm run lint` — must pass
  - [ ] Run `cd frontend && npm run type-check` — must pass

## Dev Notes

### Dependencies

- **Story 9.3 (wave 10):** Backend CRUD API for notification configs — `GET/POST /projects/{projectId}/notifications`, `PUT/DELETE /projects/{projectId}/notifications/{id}` must exist before this frontend story is functional. The test endpoint `POST /notifications/{id}/test` is added to the OpenAPI spec in this story (backend implementation is in Story 9.3 scope if not already included).
- **openapi-fetch types:** `NotificationConfig` schema is defined in Story 9.3's OpenAPI spec update — regenerate types after merging 9.3

### Architecture Requirements

Component hierarchy:

```
NotificationSettingsView.vue
├── AddChannelDialog.vue   (conditional: admin only)
├── [Skeleton rows]        (while isLoading)
├── EmptyState.vue         (when configs empty + not loading)
└── NotificationChannelRow.vue  x N
```

`useNotifications` composable holds all state and API calls. The view is a pure coordinator.

`isAdmin` check: read from auth store (`useAuthStore().user?.role === 'admin'`). Non-admin sees read-only list.

Route nesting: `/projects/:id/settings/notifications` — same `ProjectDetailView` shell. Add a "Settings" tab or a sub-navigation. Prefer adding a "Settings" tab that renders a settings layout with a sub-nav for "Notifications", "General" (future). For MVP, the Notifications route can live directly as a top-level project tab labeled "Notifications" if a full settings layout is out of scope.

### File Paths (exact)

```
api/openapi.yaml                                                        (extend: POST /notifications/{id}/test endpoint)
frontend/src/views/NotificationSettingsView.vue                         (new)
frontend/src/views/ProjectDetailView.vue                                (extend: add Notifications tab)
frontend/src/router/index.ts                                            (extend: add project-notifications child route)
frontend/src/composables/useNotifications.ts                            (new)
frontend/src/composables/__tests__/useNotifications.spec.ts             (new)
frontend/src/features/notifications/AddChannelDialog.vue                (new)
frontend/src/features/notifications/NotificationChannelRow.vue          (new)
frontend/src/features/notifications/__tests__/AddChannelDialog.spec.ts  (new, zod schema tests)
frontend/src/utils/maskUrl.ts                                           (new)
```

### Technical Specifications

**useNotifications composable:**
```typescript
export function useNotifications(projectId: string) {
  const configs = ref<NotificationConfig[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchConfigs() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiErr } = await apiClient.GET(
        '/api/v1/projects/{projectId}/notifications',
        { params: { path: { projectId } } }
      )
      if (apiErr) throw apiErr
      configs.value = data?.data ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load channels'
    } finally {
      isLoading.value = false
    }
  }

  async function createConfig(payload: CreateNotificationConfigRequest): Promise<NotificationConfig | null> {
    const { data, error: apiErr } = await apiClient.POST(
      '/api/v1/projects/{projectId}/notifications',
      { params: { path: { projectId } }, body: payload }
    )
    if (apiErr || !data) return null
    configs.value.push(data)
    return data
  }

  async function toggleEnabled(config: NotificationConfig) {
    const original = config.enabled
    // optimistic update
    const idx = configs.value.findIndex(c => c.id === config.id)
    if (idx >= 0) configs.value[idx] = { ...config, enabled: !original }
    const { error: apiErr } = await apiClient.PUT(
      '/api/v1/projects/{projectId}/notifications/{id}',
      { params: { path: { projectId, id: config.id } }, body: { enabled: !original } }
    )
    if (apiErr) {
      // revert
      if (idx >= 0) configs.value[idx] = { ...config, enabled: original }
      throw apiErr
    }
  }

  async function deleteConfig(id: string) {
    await apiClient.DELETE(
      '/api/v1/projects/{projectId}/notifications/{id}',
      { params: { path: { projectId, id } } }
    )
    configs.value = configs.value.filter(c => c.id !== id)
  }

  async function testConfig(id: string) {
    const { error: apiErr } = await apiClient.POST(
      '/api/v1/projects/{projectId}/notifications/{id}/test',
      { params: { path: { projectId, id } } }
    )
    if (apiErr) throw apiErr
  }

  onMounted(fetchConfigs)

  return { configs, isLoading, error, fetchConfigs, createConfig, toggleEnabled, deleteConfig, testConfig }
}
```

**Zod schema for Add Channel form:**
```typescript
const addChannelSchema = toTypedSchema(
  z.object({
    channel_type: z.enum(['discord', 'webhook']),
    url: z.string()
      .min(1, 'URL is required')
      .startsWith('https://', 'URL must start with https://'),
    events_filter: z.array(z.string()).default([]),
    enabled: z.boolean().default(true),
  })
)
```

**maskUrl utility:**
```typescript
export function maskUrl(url: string): string {
  if (url.length <= 6) return url
  return `****${url.slice(-6)}`
}
```

**Router addition:**
```typescript
{
  path: 'settings/notifications',
  name: 'project-notifications',
  component: () => import('@/views/NotificationSettingsView.vue'),
},
```

**ProjectDetailView tab addition:**
```typescript
{ label: 'Notifications', icon: 'pi pi-bell', route: 'project-notifications' }
```

And update `activeIndex` computed to handle the new tab.

**Event options for MultiSelect:**
```typescript
const EVENT_OPTIONS = [
  { label: 'Run Completed',          value: 'run.completed' },
  { label: 'Run Failed',             value: 'run.failed' },
  { label: 'Approval Pending',       value: 'hitl_gate.pending' },
  { label: 'Circuit Breaker Triggered', value: 'circuit_breaker.triggered' },
]
```

**ConfirmDialog usage (delete):**
```typescript
import { useConfirm } from 'primevue/useconfirm'
const confirm = useConfirm()

function handleDelete(config: NotificationConfig) {
  confirm.require({
    message: `Delete ${config.channel_type} channel?`,
    header: 'Confirm Delete',
    icon: 'pi pi-trash',
    acceptSeverity: 'danger',
    accept: async () => {
      await notifs.deleteConfig(config.id)
      toast.add({ severity: 'success', summary: 'Channel deleted', life: 3000 })
    },
  })
}
```

### Testing Requirements

**useNotifications unit tests:**
- `fetchConfigs()` populates `configs.value`, sets `isLoading` correctly
- `createConfig()` pushes to `configs.value` on success, returns null on error
- `toggleEnabled()` optimistically updates, reverts on API error
- `deleteConfig()` removes item from `configs.value`
- `testConfig()` resolves on success, throws on error

**AddChannelDialog schema tests:**
- Valid Discord URL (`https://discord.com/...`) → passes
- `http://` URL → fails with "URL must start with https://"
- Empty URL → fails with "URL is required"
- Empty `events_filter` → valid (array default)

**maskUrl tests:**
- `maskUrl('https://discord.com/api/webhooks/123/abc123')` → `'****c123'` (last 6)
- `maskUrl('https')` → `'https'` (shorter than 6, returned as-is)

### References

- Existing dialog pattern: `frontend/src/features/board/CreateStoryDialog.vue` (vee-validate + zod)
- Toast service usage: `frontend/src/views/` — existing Toast usage patterns
- ConfirmDialog: `frontend/src/ui/composed/ConfirmDialog.vue` (if exists) or `useConfirm()` from PrimeVue
- Auth store: `frontend/src/stores/auth.ts` — check `user.role` field
- PrimeVue MultiSelect: https://primevue.org/multiselect/
- PrimeVue ToggleSwitch: https://primevue.org/toggleswitch/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
