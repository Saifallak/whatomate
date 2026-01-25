import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Loader2,
  Save,
  Store,
  Globe,
  Mail,
  MapPin,
  Image as ImageIcon,
  AlertTriangle
} from 'lucide-vue-next'
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/components/ui/alert'

interface Props {
  open: boolean
  accountId: string | null
  accountName: string
}

const props = defineProps<Props>()
const emit = defineEmits(['update:open'])

interface BusinessProfile {
  messaging_product: string
  address: string
  description: string
  vertical: string
  email: string
  websites: string[]
  profile_picture_url: string
  about: string
}

const isLoading = ref(false)
const isSubmitting = ref(false)
const profile = ref<BusinessProfile>({
  messaging_product: 'whatsapp',
  address: '',
  description: '',
  vertical: '',
  email: '',
  websites: ['', ''],
  profile_picture_url: '',
  about: ''
})

// Categories (Verticals) supported by Meta
const verticals = [
  'ALCOHOL', 'APPAREL', 'AUTO', 'BEAUTY', 'EDU', 'ENTERTAIN', 'EVENT_PLAN',
  'FINANCE', 'GOVT', 'GROCERY', 'HEALTH', 'HOTEL', 'NONPROFIT',
  'ONLINE_GAMBLING', 'OTC_DRUGS', 'OTHER', 'PHYSICAL_GAMBLING',
  'PROF_SERVICES', 'RETAIL', 'TRAVEL'
]

watch(() => props.open, async (isOpen) => {
  if (isOpen && props.accountId) {
    await fetchProfile()
  }
})

async function fetchProfile() {
  if (!props.accountId) return

  isLoading.value = true
  try {
    const response = await api.get(`/accounts/${props.accountId}/business_profile`)
    const data = response.data.data
    
    // Fill the form, ensure arrays have data
    profile.value = {
      messaging_product: data.messaging_product || 'whatsapp',
      address: data.address || '',
      description: data.description || '',
      vertical: data.vertical || '',
      email: data.email || '',
      websites: data.websites && data.websites.length > 0 ? [...data.websites, ''] : ['', ''],
      profile_picture_url: data.profile_picture_url || '',
      about: data.about || ''
    }
    
    // Ensure at least two slots for websites
    if (profile.value.websites.length < 2) {
      profile.value.websites.push('')
    }
    // Trim to max 2
    profile.value.websites = profile.value.websites.slice(0, 2)
    
  } catch (error: any) {
    console.error('Failed to fetch business profile:', error)
    toast.error('Failed to load business profile')
  } finally {
    isLoading.value = false
  }
}

async function saveProfile() {
  if (!props.accountId) return

  isSubmitting.value = true
  try {
    // Filter out empty websites
    const websites = profile.value.websites.filter(w => w.trim() !== '')

    const payload = {
      messaging_product: 'whatsapp',
      address: profile.value.address,
      description: profile.value.description,
      vertical: profile.value.vertical,
      email: profile.value.email,
      websites: websites,
      about: profile.value.about
      // Note: profile_picture_handle is handled separately effectively by the user having to upload first (not implemented in this specific form yet)
    }

    await api.put(`/accounts/${props.accountId}/business_profile`, payload)
    toast.success('Business profile updated successfully')
    emit('update:open', false)
  } catch (error: any) {
    console.error('Failed to update profile:', error)
    const message = error.response?.data?.message || 'Failed to update profile'
    toast.error(message)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="max-w-2xl max-h-[90vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Store class="h-5 w-5 text-emerald-500" />
          Business Profile: {{ accountName }}
        </DialogTitle>
        <DialogDescription>
          Update your WhatsApp Business profile details. These are visible to your customers.
        </DialogDescription>
      </DialogHeader>

      <div v-if="isLoading" class="py-12 flex justify-center">
        <Loader2 class="h-8 w-8 animate-spin text-muted-foreground" />
      </div>

      <div v-else class="space-y-6 py-4">
        <!-- Warning about Name Review (Static for now as name update isn't directly exposed here yet, but context is important) -->
        <Alert variant="warning" class="bg-amber-950/20 border-amber-900/50 text-amber-600 dark:text-amber-400">
          <AlertTriangle class="h-4 w-4" />
          <AlertTitle>Profile Updates</AlertTitle>
          <AlertDescription>
            Changes to your address, description, email, and websites usually update immediately.
            <br/>Note: Updating the Business Display Name (not available here) triggers a Meta review process.
          </AlertDescription>
        </Alert>

        <div class="grid gap-6 md:grid-cols-2">
           <!-- Profile Picture Preview -->
           <div class="md:col-span-2 flex items-center gap-4">
              <div class="h-20 w-20 rounded-full bg-secondary flex items-center justify-center overflow-hidden border border-border">
                <img v-if="profile.profile_picture_url" :src="profile.profile_picture_url" alt="Profile" class="h-full w-full object-cover" />
                <ImageIcon v-else class="h-8 w-8 text-muted-foreground" />
              </div>
              <div>
                <Label>Profile Picture</Label>
                <p class="text-xs text-muted-foreground mt-1">
                  Profile picture updates require a simpler media upload flow. 
                  <br/>Currently, this can be updated via the Meta Business Manager directly.
                </p>
              </div>
           </div>

           <!-- About -->
           <div class="md:col-span-2 space-y-2">
            <Label for="about">About (Status)</Label>
            <Input id="about" v-model="profile.about" placeholder="e.g., Available, Busy, At work" :maxlength="139" />
            <p class="text-xs text-muted-foreground text-right">{{ profile.about.length }}/139</p>
          </div>

          <!-- Description -->
          <div class="md:col-span-2 space-y-2">
            <Label for="description">Business Description</Label>
            <Textarea id="description" v-model="profile.description" placeholder="Describe your business..." rows="3" :maxlength="512" />
            <p class="text-xs text-muted-foreground text-right">{{ profile.description.length }}/512</p>
          </div>

          <!-- Vertical (Category) -->
          <div class="space-y-2">
            <Label for="vertical">Industry (Vertical)</Label>
            <select
              id="vertical"
              v-model="profile.vertical"
              class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            >
              <option value="" disabled>Select a category</option>
              <option v-for="v in verticals" :key="v" :value="v">{{ v }}</option>
            </select>
          </div>

           <!-- Email -->
           <div class="space-y-2">
            <Label for="email">Contact Email</Label>
            <div class="relative">
              <Mail class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input id="email" v-model="profile.email" type="email" class="pl-9" placeholder="contact@example.com" :maxlength="128" />
            </div>
          </div>

          <!-- Address -->
          <div class="md:col-span-2 space-y-2">
            <Label for="address">Business Address</Label>
            <div class="relative">
              <MapPin class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input id="address" v-model="profile.address" class="pl-9" placeholder="Street, City, State, Zip" :maxlength="256" />
            </div>
          </div>

          <!-- Websites -->
          <div class="md:col-span-2 space-y-3">
             <Label>Websites (Max 2)</Label>
             <div v-for="(_, index) in profile.websites" :key="index" class="relative">
               <Globe class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
               <Input v-model="profile.websites[index]" class="pl-9" placeholder="https://www.example.com" :maxlength="256" />
             </div>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="$emit('update:open', false)">Cancel</Button>
        <Button @click="saveProfile" :disabled="isSubmitting || isLoading">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 mr-2 animate-spin" />
          <Save v-else class="h-4 w-4 mr-2" />
          Save Changes
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
