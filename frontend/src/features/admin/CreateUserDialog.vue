<script setup lang="ts">
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Select from 'primevue/select'
import Button from 'primevue/button'
import { useUsers } from '@/composables/useUsers'
import { watch } from 'vue'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  created: []
}>()

const { createUser } = useUsers()

const createUserSchema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    name: z.string().min(1, 'Name is required').max(255, 'Name too long'),
    role: z.enum(['admin', 'member']),
  }),
)

const { handleSubmit, resetForm } = useForm({
  validationSchema: createUserSchema,
  initialValues: { email: '', password: '', name: '', role: 'member' as const },
})

const { value: email, errorMessage: emailError } = useField<string>('email')
const { value: password, errorMessage: passwordError } = useField<string>('password')
const { value: name, errorMessage: nameError } = useField<string>('name')
const { value: role, errorMessage: roleError } = useField<string>('role')

const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Member', value: 'member' },
]

watch(
  () => props.visible,
  (val) => {
    if (val) resetForm()
  },
)

const onSubmit = handleSubmit(async (values) => {
  await createUser.execute(values)
  if (!createUser.error.value) {
    emit('created')
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
    header="Create User"
    modal
    :style="{ width: '28rem' }"
    @update:visible="emit('update:visible', $event)"
  >
    <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
      <div class="flex flex-col gap-1">
        <label for="create-name" class="text-sm font-medium">Name</label>
        <InputText id="create-name" v-model="name" :invalid="!!nameError" />
        <small v-if="nameError" class="text-red-500">{{ nameError }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <label for="create-email" class="text-sm font-medium">Email</label>
        <InputText id="create-email" v-model="email" type="email" :invalid="!!emailError" />
        <small v-if="emailError" class="text-red-500">{{ emailError }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <label for="create-password" class="text-sm font-medium">Password</label>
        <Password
          id="create-password"
          v-model="password"
          :feedback="false"
          toggle-mask
          :invalid="!!passwordError"
          input-class="w-full"
          class="w-full"
        />
        <small v-if="passwordError" class="text-red-500">{{ passwordError }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <label for="create-role" class="text-sm font-medium">Role</label>
        <Select
          id="create-role"
          v-model="role"
          :options="roleOptions"
          option-label="label"
          option-value="value"
          :invalid="!!roleError"
        />
        <small v-if="roleError" class="text-red-500">{{ roleError }}</small>
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <Button type="button" label="Cancel" severity="secondary" text @click="onCancel" />
        <Button type="submit" label="Create" :loading="createUser.isLoading.value" />
      </div>
    </form>
  </Dialog>
</template>
