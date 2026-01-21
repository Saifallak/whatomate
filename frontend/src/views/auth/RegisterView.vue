<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'vue-sonner'
import { Loader2 } from 'lucide-vue-next'
import { appConfig } from '@/config/app'

const router = useRouter()
const authStore = useAuthStore()

const fullName = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const organizationName = ref('')
const isLoading = ref(false)

const handleRegister = async () => {
  if (!fullName.value || !email.value || !password.value || !organizationName.value) {
    toast.error('Please fill in all fields')
    return
  }

  if (password.value !== confirmPassword.value) {
    toast.error('Passwords do not match')
    return
  }

  if (password.value.length < 8) {
    toast.error('Password must be at least 8 characters')
    return
  }

  isLoading.value = true

  try {
    await authStore.register({
      full_name: fullName.value,
      email: email.value,
      password: password.value,
      organization_name: organizationName.value
    })
    toast.success('Registration successful')
    router.push('/')
  } catch (error: any) {
    const message = error.response?.data?.message || 'Registration failed'
    toast.error(message)
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-[#0a0a0b] light:bg-gradient-to-br light:from-gray-50 light:to-gray-100 p-4">
    <div class="w-full max-w-md rounded-2xl border border-white/[0.08] bg-white/[0.02] backdrop-blur light:bg-white light:border-gray-200 light:shadow-xl">
      <div class="p-8 space-y-1 text-center">
        <div class="flex justify-center mb-4">
          <img :src="appConfig.icon" :alt="appConfig.name + ' logo'" class="h-12 w-12" />
        </div>
        <h2 class="text-2xl font-bold text-white light:text-gray-900">Create an account</h2>
        <p class="text-white/50 light:text-gray-500">
          Start your WhatsApp Business journey with {{ appConfig.name }}
        </p>
      </div>
      <form @submit.prevent="handleRegister">
        <div class="px-8 pb-4 space-y-4">
          <div class="space-y-2">
            <Label for="fullName">Full Name</Label>
            <Input
              id="fullName"
              v-model="fullName"
              type="text"
              placeholder="John Doe"
              :disabled="isLoading"
              autocomplete="name"
            />
          </div>
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input
              id="email"
              v-model="email"
              type="email"
              placeholder="name@example.com"
              :disabled="isLoading"
              autocomplete="email"
            />
          </div>
          <div class="space-y-2">
            <Label for="organizationName">Organization Name</Label>
            <Input
              id="organizationName"
              v-model="organizationName"
              type="text"
              placeholder="Your Company"
              :disabled="isLoading"
            />
          </div>
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="At least 8 characters"
              :disabled="isLoading"
              autocomplete="new-password"
            />
          </div>
          <div class="space-y-2">
            <Label for="confirmPassword">Confirm Password</Label>
            <Input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              placeholder="Confirm your password"
              :disabled="isLoading"
              autocomplete="new-password"
            />
          </div>
        </div>
        <div class="px-8 pb-8">
          <Button type="submit" class="w-full mb-4" :disabled="isLoading">
            <Loader2 v-if="isLoading" class="mr-2 h-4 w-4 animate-spin" />
            Create account
          </Button>
          <p class="text-sm text-center text-white/40 light:text-gray-500">
            Already have an account?
            <RouterLink to="/login" class="text-primary hover:underline">
              Sign in
            </RouterLink>
          </p>
        </div>
      </form>
    </div>
  </div>
</template>
