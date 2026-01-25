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
  // Extract {{1}}, {{Variable}} etc from body_content
  const matches = currentTemplate.value.body_content?.match(/\{\{([^}]+)\}\}/g) || []
  // Strip braces for the key
  return ([...new Set(matches)] as string[]).map(m => m.replace(/\{\{|\}\}/g, '')).sort()
})

// Watch account change to fetch templates
import { watch } from 'vue'
watch(selectedAccount, async (newAccountId) => {
  if (!newAccountId) {
    templates.value = []
    return
  }
  
  // Find the account object to get the name
  const account = accounts.value.find(a => a.id === newAccountId)
  if (!account) return

  isLoading.value = true
  try {
    // API appears to expect Account Name for filtering templates, not ID
    const response = await templatesService.list({ 
        status: 'APPROVED',
        account: account.name 
    })
    const data = response.data.data || response.data
    templates.value = data.templates || data || []
  } catch (error) {
    console.error('Failed to load templates:', error)
    toast.error('Failed to load templates')
  } finally {
    isLoading.value = false
  }
})

async function fetchData() {
  isLoading.value = true
  try {
    const response = await accountsService.list()
    const data = response.data.data || response.data
    accounts.value = data.accounts || []
    
    if (accounts.value.length > 0) {
      // Default to first account
      selectedAccount.value = accounts.value[0].id || accounts.value[0].whatsapp_account_id
    }
  } catch (error) {
    console.error('Failed to load accounts:', error)
    toast.error('Failed to load accounts')
    isLoading.value = false
  }
}


const duplicateWarning = ref<string | null>(null)

// Watch phone number to check for existing contacts on other accounts
// Manual debounce used below

let checkTimeout: any = null

const checkExistingContacts = async (phone: string) => {
    if (!phone || phone.length < 5) { // Allow check for shorter strings for search-as-you-type
        duplicateWarning.value = null
        return
    }
    
    try {
        const res = await contactsService.list({ search: phone }) // Use search for LIKE match
        const contacts = res.data.data?.contacts || res.data.contacts || []
        
        const account = accounts.value.find(a => a.id === selectedAccount.value)
        const currentAccountName = account?.name
        
        // Filter to ensure we are matching phone number (search also checks name)
        // and allow partial matches for "search-as-you-type" warning
        // but prioritize exact match if available
        const matchingContacts = contacts.filter((c: any) => c.phone_number.includes(phone))
        
        const otherAccountContact = matchingContacts.find((c: any) => c.whatsapp_account !== currentAccountName)
        
        if (otherAccountContact) {
            duplicateWarning.value = `Contact "${otherAccountContact.name || otherAccountContact.phone_number}" already exists on account "${otherAccountContact.whatsapp_account}".`
        } else {
            duplicateWarning.value = null
        }
    } catch (e) {
        // ignore
    }
}

watch(phoneNumber, (newVal) => {
    if (checkTimeout) clearTimeout(checkTimeout)
    checkTimeout = setTimeout(() => {
        checkExistingContacts(newVal)
    }, 500)
})


async function handleSubmit() {
  if (!phoneNumber.value || !selectedTemplate.value || !selectedAccount.value) return

  isSubmitting.value = true
  try {
    // Get Account Name
    const account = accounts.value.find(a => a.id === selectedAccount.value)
    if (!account) throw new Error('Selected account not found')
    
    // 1. Create Contact
    let contactId = ''
    try {
        const createRes = await contactsService.create({
            phone_number: phoneNumber.value,
            name: contactName.value || phoneNumber.value,
            whatsapp_account: account.name // Pass Name, not ID
        })
        const contactData = createRes.data.data || createRes.data
        contactId = contactData.id
    } catch (error: any) {
         if (error.response?.status === 409) {
             // Should not happen with new backend logic unless strictly prevented, 
             // but if it does, it implies we can't create. 
             // Currently backend returns 200/201 if success/exists-same-account.
             throw new Error('Contact already exists or could not be created.')
         }
         throw error
    }

    // 2. Send Template Message
    await messagesService.sendTemplate(contactId, {
        template_name: selectedTemplate.value,
        template_params: templateParams.value,
        account_name: account.name // Pass Account Name
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

watch(() => props.open, (newVal) => {
    if (newVal) fetchData()
})
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Start New Conversation</DialogTitle>
        <DialogDescription>
          Create a contact and send a template to start chatting.
        </DialogDescription>
      </DialogHeader>

      <div v-if="isLoading" class="flex justify-center py-8">
        <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
      </div>

      <div v-else class="grid gap-4 py-4">
        <!-- Account Selection -->
        <div class="grid gap-2">
          <Label>WhatsApp Account <span class="text-red-500">*</span></Label>
          <Select v-model="selectedAccount">
            <SelectTrigger>
              <SelectValue placeholder="Select account" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="acc in accounts" :key="acc.id" :value="acc.id">
                {{ acc.name }} ({{ acc.phone_number || acc.phone_id }})
              </SelectItem>
              <div v-if="accounts.length === 0" class="p-2 text-sm text-muted-foreground text-center">
                No accounts found
              </div>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-2">
          <Label>Phone Number <span class="text-red-500">*</span></Label>
          <Input 
            v-model="phoneNumber" 
            placeholder="+1234567890" 
          />
          <div v-if="duplicateWarning" class="text-xs text-amber-600 bg-amber-50 p-2 rounded border border-amber-200 mt-1">
            <span class="font-medium">Notice:</span> {{ duplicateWarning }}
          </div>
        </div>

        <div class="grid gap-2">
          <Label>Name (Optional)</Label>
          <Input 
            v-model="contactName" 
            placeholder="John Doe" 
          />
        </div>

        <div class="grid gap-2">
          <Label>Template <span class="text-red-500">*</span></Label>
          <Select v-model="selectedTemplate">
            <SelectTrigger>
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
        <div v-if="templatePlaceholders.length > 0" class="grid gap-2 border-t pt-2 mt-2">
          <Label class="text-xs text-muted-foreground">Template Variables</Label>
          <div v-for="placeholder in templatePlaceholders" :key="placeholder" class="grid grid-cols-4 items-center gap-2">
            <Label class="text-right text-xs">{{ '{' + placeholder + '}' }}</Label>
            <Input 
              v-model="templateParams[placeholder]" 
              class="col-span-3 h-8 text-xs" 
              :placeholder="`Value for ${placeholder}`"
            />
          </div>
        </div>

        <div class="text-xs text-muted-foreground italic" v-if="currentTemplate">
          Preview: {{ currentTemplate.body_content }}
        </div>
      </div>

      <DialogFooter>
        <Button variant="ghost" @click="$emit('update:open', false)">Cancel</Button>
        <Button 
            @click="handleSubmit" 
            :disabled="!phoneNumber || !selectedTemplate || !selectedAccount || isSubmitting"
            class="bg-emerald-600 hover:bg-emerald-700 text-white"
        >
          <Loader2 v-if="isSubmitting" class="mr-2 h-4 w-4 animate-spin" />
          Start Chat
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
