<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Loader2 } from 'lucide-vue-next'
import { contactsService, templatesService, accountsService, messagesService } from '@/services/api'
import { toast } from 'vue-sonner'
import { useContactsStore } from '@/stores/contacts'

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits(['update:open', 'chat-created'])

const contactsStore = useContactsStore()

const isLoading = ref(false)
const isSubmitting = ref(false)

// Form Data
const phoneNumber = ref('')
const contactName = ref('')
const selectedAccount = ref('')
const selectedTemplate = ref('')
const templateParams = ref<Record<string, string>>({})

// Data Sources
const accounts = ref<any[]>([])
const templates = ref<any[]>([])

// Computed
const currentTemplate = computed(() => {
  return templates.value.find(t => t.name === selectedTemplate.value)
})

const templatePlaceholders = computed(() => {
  if (!currentTemplate.value) return []
  // Extract {{1}}, {{2}} etc from body_content
  const matches = currentTemplate.value.body_content?.match(/\{\{\d+\}\}/g) || []
  return [...new Set(matches)].sort()
})

async function fetchData() {
  isLoading.value = true
  try {
    const [accRes, tplRes] = await Promise.all([
      accountsService.list(),
      templatesService.list({ status: 'APPROVED' })
    ])
    
    const accData = accRes.data.data || accRes.data
    accounts.value = accData.accounts || []
    if (accounts.value.length > 0) {
      selectedAccount.value = accounts.value[0].whatsapp_account_id || accounts.value[0].phone_number_id || '' // Adjust based on actual API response structure
    }

    const tplData = tplRes.data.data || tplRes.data
    templates.value = tplData.templates || tplData || []
  } catch (error) {
    console.error('Failed to load data:', error)
    toast.error('Failed to load accounts or templates')
  } finally {
    isLoading.value = false
  }
}

async function handleSubmit() {
  if (!phoneNumber.value || !selectedTemplate.value) return

  isSubmitting.value = true
  try {
    // 1. Create Contact (if not exists logic handled by backend usually, or we just try create)
    // We'll try to create, if it fails with "exists", we fetch it (simplified: just try create/get)
    let contactId = ''
    
    // Check if contact exists locally first to save a call? No, assume we need to ensure backend has it.
    try {
        const createRes = await contactsService.create({
            phone_number: phoneNumber.value,
            name: contactName.value || phoneNumber.value,
            whatsapp_account: selectedAccount.value
        })
        const contactData = createRes.data.data || createRes.data
        contactId = contactData.id
    } catch (error: any) {
        // If 409 conflict, it means contact exists. We should search for it.
        // For now, let's assume create works or returns existing if logic permits. 
        // If strict duplicate check fails, we'd need to Search-and-Select. 
        // But for "New Chat" button, let's assume valid flow.
        if (error.response?.status === 409) {
           // Fallback: search? 
           // For prototype, let's just toast error if duplicate, user should use search.
           // OR: intelligent backend 'get_or_create' is best.
           // Let's assume error for now.
           throw new Error('Contact already exists. Please search for them instead.')
        }
        throw error
    }

    // 2. Send Template Message
    await messagesService.sendTemplate(contactId, {
        template_name: selectedTemplate.value,
        template_params: templateParams.value
    })

    toast.success('Conversation started!')
    emit('update:open', false)
    emit('chat-created', contactId)
    
    // Refresh contacts and select new one
    await contactsStore.fetchContacts()
    contactsStore.fetchContact(contactId) // Select it
    
  } catch (error: any) {
    console.error('Failed to create chat:', error)
    toast.error(error.message || 'Failed to start conversation')
  } finally {
    isSubmitting.value = false
  }
}

onMounted(() => {
    if (props.open) fetchData()
})
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-[425px] bg-[#1c1c1e] text-white border-white/10">
      <DialogHeader>
        <DialogTitle>Start New Conversation</DialogTitle>
        <DialogDescription>
          Create a contact and send a template to start chatting.
        </DialogDescription>
      </DialogHeader>

      <div v-if="isLoading" class="flex justify-center py-8">
        <Loader2 class="h-8 w-8 animate-spin text-white/50" />
      </div>

      <div v-else class="grid gap-4 py-4">
        <!-- Account Selection (only if multiple?) -->
        <div class="grid gap-2" v-if="accounts.length > 1">
          <Label>WhatsApp Account</Label>
          <Select v-model="selectedAccount">
            <SelectTrigger class="bg-white/5 border-white/10 text-white">
              <SelectValue placeholder="Select account" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="acc in accounts" :key="acc.id" :value="acc.id">
                {{ acc.name }} ({{ acc.phone_number }})
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-2">
          <Label>Phone Number</Label>
          <Input 
            v-model="phoneNumber" 
            placeholder="+1234567890" 
            class="bg-white/5 border-white/10 text-white placeholder:text-white/30"
          />
        </div>

        <div class="grid gap-2">
          <Label>Name (Optional)</Label>
          <Input 
            v-model="contactName" 
            placeholder="John Doe" 
            class="bg-white/5 border-white/10 text-white placeholder:text-white/30"
          />
        </div>

        <div class="grid gap-2">
          <Label>Template</Label>
          <Select v-model="selectedTemplate">
            <SelectTrigger class="bg-white/5 border-white/10 text-white">
              <SelectValue placeholder="Select a template" />
            </SelectTrigger>
            <SelectContent class="max-h-[200px]">
              <SelectItem v-for="tpl in templates" :key="tpl.name" :value="tpl.name">
                {{ tpl.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <!-- Dynamic Template Params -->
        <div v-if="templatePlaceholders.length > 0" class="grid gap-2 border-t border-white/10 pt-2 mt-2">
          <Label class="text-xs text-white/50">Template Variables</Label>
          <div v-for="placeholder in templatePlaceholders" :key="placeholder" class="grid grid-cols-4 items-center gap-2">
            <Label class="text-right text-xs">{{ placeholder }}</Label>
            <Input 
              v-model="templateParams[placeholder]" 
              class="col-span-3 h-8 text-xs bg-white/5 border-white/10 text-white" 
              :placeholder="`Value for ${placeholder}`"
            />
          </div>
        </div>

        <div class="text-xs text-white/40 italic" v-if="currentTemplate">
          Preview: {{ currentTemplate.body_content }}
        </div>
      </div>

      <DialogFooter>
        <Button variant="ghost" @click="$emit('update:open', false)">Cancel</Button>
        <Button 
            @click="handleSubmit" 
            :disabled="!phoneNumber || !selectedTemplate || isSubmitting"
            class="bg-emerald-600 hover:bg-emerald-700 text-white"
        >
          <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
          Start Chat
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
