<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Loader2, Send, FileText } from 'lucide-vue-next'
import { api } from '@/services/api'
import { useContactsStore } from '@/stores/contacts'
import { toast } from 'vue-sonner'
import DOMPurify from 'dompurify'

const props = defineProps<{
  open: boolean
  contactId: string
  whatsappAccount?: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'sent'): void
}>()

const contactsStore = useContactsStore()

interface Template {
  id: string
  name: string
  display_name: string
  language: string
  category: string
  status: string
  header_type: string
  header_content: string
  body_content: string
  footer_content: string
  whatsapp_account: string
}

const templates = ref<Template[]>([])
const isLoading = ref(false)
const isSending = ref(false)
const selectedTemplateId = ref<string>('')
const variableValues = ref<Record<string, string>>({})
const headerVariableValues = ref<Record<string, string>>({})

const isOpen = computed({
  get: () => props.open,
  set: (val) => emit('update:open', val)
})

const selectedTemplate = computed(() => 
  templates.value.find(t => t.id === selectedTemplateId.value)
)

// Fetch templates when dialog opens
watch(() => props.open, async (newVal) => {
  if (newVal) {
    selectedTemplateId.value = ''
    variableValues.value = {}
    headerVariableValues.value = {}
    await fetchTemplates()
  }
})

async function fetchTemplates() {
  isLoading.value = true
  try {
    const params = props.whatsappAccount ? `?account=${props.whatsappAccount}&status=APPROVED` : '?status=APPROVED'
    const response = await api.get(`/templates${params}`)
    templates.value = response.data.data?.templates || []
  } catch (error) {
    console.error('Failed to fetch templates:', error)
    toast.error('Failed to load templates')
  } finally {
    isLoading.value = false
  }
}

// Extract variables when template is selected
watch(selectedTemplate, (template) => {
  if (!template) return
  
  // Reset values
  variableValues.value = {}
  headerVariableValues.value = {}
  
  // Extract body variables {{1}}, {{2}} or {{name}}
  const bodyMatches = template.body_content.match(/\{\{([^}]+)\}\}/g) || []
  bodyMatches.forEach(match => {
    const name = match.replace(/[{}]/g, '').trim()
    variableValues.value[name] = ''
  })
  
  // Extract header variables if text header
  if (template.header_type === 'TEXT' && template.header_content) {
    const headerMatches = template.header_content.match(/\{\{([^}]+)\}\}/g) || []
    headerMatches.forEach(match => {
      const name = match.replace(/[{}]/g, '').trim()
      headerVariableValues.value[name] = ''
    })
  }
})

function formatPreview(text: string, values: Record<string, string>): string {
  if (!text) return ''
  let result = DOMPurify.sanitize(text, { ALLOWED_TAGS: [] })
  
  Object.entries(values).forEach(([key, value]) => {
    const placeholder = value || `{{${key}}}`
    const className = value ? 'bg-green-100 dark:bg-green-900 px-1 rounded' : 'bg-yellow-100 dark:bg-yellow-900 px-1 rounded'
    result = result.replace(new RegExp(`\\{\\{${key}\\}\\}`, 'g'), `<span class="${className}">${placeholder}</span>`)
  })
  
  return result
}

async function sendTemplate() {
  if (!selectedTemplate.value) return
  
  // Validate all variables are filled
  const missingBody = Object.entries(variableValues.value).some(([_, val]) => !val.trim())
  const missingHeader = Object.entries(headerVariableValues.value).some(([_, val]) => !val.trim())
  
  if (missingBody || missingHeader) {
    toast.error('Please fill in all variables')
    return
  }
  
  isSending.value = true
  try {
    // Build template_params - merge header and body variables
    const templateParams: Record<string, string> = {
      ...headerVariableValues.value,
      ...variableValues.value
    }
    
    await contactsStore.sendTemplate(
      props.contactId,
      selectedTemplate.value.name,
      templateParams
    )
    
    toast.success('Template sent successfully')
    isOpen.value = false
    emit('sent')
  } catch (error: any) {
    const message = error.response?.data?.message || 'Failed to send template'
    toast.error(message)
  } finally {
    isSending.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="max-w-md max-h-[90vh] flex flex-col p-0">
      <DialogHeader class="p-6 pb-2">
        <DialogTitle>Send Template Message</DialogTitle>
        <DialogDescription>
          Pick a template to start a conversation or re-engage with the customer.
        </DialogDescription>
      </DialogHeader>

      <div v-if="isLoading" class="flex-1 flex items-center justify-center min-h-[200px]">
        <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
      </div>

      <div v-else class="flex-1 overflow-hidden flex flex-col">
        <div class="p-6 pt-2 pb-0">
            <Label>Select Template</Label>
            <Select v-model="selectedTemplateId">
                <SelectTrigger class="w-full mt-1.5">
                    <SelectValue placeholder="Select a template..." />
                </SelectTrigger>
                <SelectContent>
                    <SelectItem v-for="t in templates" :key="t.id" :value="t.id">
                        {{ t.display_name || t.name }}
                    </SelectItem>
                </SelectContent>
            </Select>
        </div>

        <ScrollArea class="flex-1 p-6">
            <div v-if="selectedTemplate" class="space-y-6">
                <!-- Preview -->
                <div class="bg-muted/50 rounded-lg p-4 space-y-3">
                    <h4 class="text-sm font-medium text-muted-foreground mb-2">Preview</h4>
                    <div v-if="selectedTemplate.header_content" class="text-sm font-semibold mb-2">
                         <span v-html="formatPreview(selectedTemplate.header_content, headerVariableValues)"></span>
                    </div>
                    <div class="text-sm whitespace-pre-wrap">
                        <span v-html="formatPreview(selectedTemplate.body_content, variableValues)"></span>
                    </div>
                    <div v-if="selectedTemplate.footer_content" class="text-xs text-muted-foreground mt-2">
                        {{ selectedTemplate.footer_content }}
                    </div>
                </div>

                <!-- Variables -->
                <div v-if="Object.keys(headerVariableValues).length > 0" class="space-y-3">
                    <h4 class="text-sm font-medium">Header Variables</h4>
                    <div v-for="(_, key) in headerVariableValues" :key="key" class="space-y-1">
                        <Label class="text-xs text-muted-foreground">Value for {{ key }}</Label>
                        <Input v-model="headerVariableValues[key]" :placeholder="`Enter value for {{${key}}}`" />
                    </div>
                </div>

                <div v-if="Object.keys(variableValues).length > 0" class="space-y-3">
                    <h4 class="text-sm font-medium">Body Variables</h4>
                    <div v-for="(_, key) in variableValues" :key="key" class="space-y-1">
                        <Label class="text-xs text-muted-foreground">Value for {{ key }}</Label>
                        <Input v-model="variableValues[key]" :placeholder="`Enter value for {{${key}}}`" />
                    </div>
                </div>
            </div>
            
            <div v-else-if="!isLoading && templates.length === 0" class="text-center py-8 text-muted-foreground">
                <FileText class="h-10 w-10 mx-auto mb-3 opacity-20" />
                <p>No approved templates found for this account.</p>
            </div>
        </ScrollArea>
      </div>

      <DialogFooter class="p-6 pt-2">
        <Button variant="outline" @click="isOpen = false">Cancel</Button>
        <Button 
            @click="sendTemplate" 
            :disabled="!selectedTemplate || isSending"
            class="bg-green-600 hover:bg-green-700 text-white"
        >
            <Loader2 v-if="isSending" class="h-4 w-4 mr-2 animate-spin" />
            <Send v-else class="h-4 w-4 mr-2" />
            Send Template
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
