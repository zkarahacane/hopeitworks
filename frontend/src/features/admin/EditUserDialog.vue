<script setup lang="ts">
import { watch } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import type { User } from '@/stores/auth'
import { useUsers } from '@/composables/useUsers'

const props = defineProps<{
  visible: boolean
  user: User | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  updated: []
}>()

const { updateUser } = useUsers()

const editUserSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Name is required').max(255, 'Name too long'),
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
  }),
)

const { handleSubmit, resetForm } = useForm({
  validationSchema: editUserSchema,
})

const { value: name, errorMessage: nameError } = useField<string>('name')
const { value: email, errorMessage: emailError } = useField<string>('email')

watch(
  () => props.user,
  (user) => {
    if (user) {
      resetForm({
        values: {
          name: user.name,
          email: user.email,
        },
      })
    }
  },
)

const onSubmit = handleSubmit(async (values) => {
  if (!props.user) return
  await updateUser.execute(props.user.id, values)
  if (!updateUser.error.value) {
    emit('updated')
    emit('update:visible', false)
  }
})

function onCancel() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    header="Edit User"
    modal
    :style="{ width: '28rem' }"
    @update:visible="emit('update:visible', $event)"
  >
    <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
      <div class="flex flex-col gap-1">
        <label for="edit-name" class="text-sm font-medium">Name</label>
        <InputText id="edit-name" v-model="name" :invalid="!!nameError" />
        <small v-if="nameError" class="text-red-500">{{ nameError }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <label for="edit-email" class="text-sm font-medium">Email</label>
        <InputText id="edit-email" v-model="email" type="email" :invalid="!!emailError" />
        <small v-if="emailError" class="text-red-500">{{ emailError }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <span class="text-sm font-medium">Role</span>
        <Tag
          v-if="user"
          :value="user.role"
          :severity="user.role === 'admin' ? 'danger' : 'info'"
        />
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" label="Cancel" severity="secondary" text @click="onCancel" />
        <Button type="submit" label="Save" :loading="updateUser.isLoading.value" />
      </div>
    </form>
  </Dialog>
</template>
