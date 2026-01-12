<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { X, ChevronDown, ChevronRight, Phone, User } from 'lucide-vue-next'
import { getInitials } from '@/lib/utils'
import { contactsService } from '@/services/api'
import type { Contact } from '@/stores/contacts'

interface PanelFieldConfig {
  key: string
  label: string
  order: number
  display_type?: 'text' | 'badge' | 'tag'
  color?: 'default' | 'success' | 'warning' | 'error' | 'info'
}

interface PanelSection {
  id: string
  label: string
  columns: number
  collapsible: boolean
  default_collapsed: boolean
  order: number
  fields: PanelFieldConfig[]
}

interface PanelConfig {
  sections: PanelSection[]
}

interface SessionData {
  session_id?: string
  flow_id?: string
  flow_name?: string
  session_data: Record<string, any>
  panel_config: PanelConfig
}

const props = defineProps<{
  contact: Contact
}>()

const emit = defineEmits<{
  close: []
}>()

const isLoading = ref(false)
const sessionData = ref<SessionData | null>(null)
const collapsedSections = ref<Record<string, boolean>>({})

// Resizable panel state
const MIN_WIDTH = 280
const MAX_WIDTH = 480
const panelWidth = ref(MAX_WIDTH) // Default to max width
const isResizing = ref(false)

function startResize(e: MouseEvent) {
  isResizing.value = true
  const startX = e.clientX
  const startWidth = panelWidth.value

  function onMouseMove(e: MouseEvent) {
    // Dragging left increases width, dragging right decreases
    const delta = startX - e.clientX
    const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, startWidth + delta))
    panelWidth.value = newWidth
  }

  function onMouseUp() {
    isResizing.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Fetch session data when contact changes
watch(() => props.contact.id, async (newId) => {
  if (newId) {
    await fetchSessionData()
  }
}, { immediate: true })

async function fetchSessionData() {
  isLoading.value = true
  try {
    const response = await contactsService.getSessionData(props.contact.id)
    sessionData.value = response.data.data || response.data

    // Initialize collapsed state based on default_collapsed
    if (sessionData.value?.panel_config?.sections) {
      for (const section of sessionData.value.panel_config.sections) {
        if (collapsedSections.value[section.id] === undefined) {
          collapsedSections.value[section.id] = section.default_collapsed
        }
      }
    }
  } catch (error) {
    console.error('Failed to fetch session data:', error)
    sessionData.value = null
  } finally {
    isLoading.value = false
  }
}

function toggleSection(sectionId: string) {
  collapsedSections.value[sectionId] = !collapsedSections.value[sectionId]
}

function isSectionCollapsed(sectionId: string): boolean {
  return collapsedSections.value[sectionId] ?? false
}

function getFieldValue(key: string): string {
  if (!sessionData.value?.session_data) return '-'
  const value = sessionData.value.session_data[key]
  if (value === undefined || value === null || value === '') return '-'
  return String(value)
}

function getColorClass(color?: string): string {
  switch (color) {
    case 'success':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'warning':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'error':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'info':
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

// Sort sections by order
const sortedSections = computed(() => {
  if (!sessionData.value?.panel_config?.sections) return []
  return [...sessionData.value.panel_config.sections].sort((a, b) => a.order - b.order)
})

// Get tags from contact
const contactTags = computed(() => {
  if (!props.contact.tags || !Array.isArray(props.contact.tags)) return []
  return props.contact.tags
})
</script>

<template>
  <div
    class="flex flex-col bg-card h-full relative"
    :style="{ width: `${panelWidth}px` }"
  >
    <!-- Resize Handle -->
    <div
      class="absolute left-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/20 active:bg-primary/30 z-10 border-l"
      :class="{ 'bg-primary/30': isResizing }"
      @mousedown="startResize"
    />

    <!-- Header -->
    <div class="h-12 px-3 border-b flex items-center justify-between">
      <h3 class="font-medium text-sm">Contact Info</h3>
      <Button variant="ghost" size="icon" class="h-8 w-8" @click="emit('close')">
        <X class="h-4 w-4" />
      </Button>
    </div>

    <ScrollArea class="flex-1">
      <div class="p-4 space-y-4">
        <!-- Contact Header -->
        <div class="flex flex-col items-center text-center pb-4 border-b">
          <Avatar class="h-16 w-16 mb-3">
            <AvatarImage :src="contact.avatar_url" />
            <AvatarFallback class="text-lg">
              {{ getInitials(contact.name || contact.phone_number) }}
            </AvatarFallback>
          </Avatar>
          <h4 class="font-medium">
            {{ contact.name || contact.phone_number }}
          </h4>
          <div class="flex items-center gap-1 text-sm text-muted-foreground mt-1">
            <Phone class="h-3 w-3" />
            <span>{{ contact.phone_number }}</span>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="isLoading" class="space-y-4">
          <Skeleton class="h-8 w-full" />
          <Skeleton class="h-20 w-full" />
          <Skeleton class="h-8 w-full" />
          <Skeleton class="h-20 w-full" />
        </div>

        <!-- No Session Data or no panel config -->
        <div v-else-if="!sessionData || sortedSections.length === 0" class="text-center py-6 text-muted-foreground">
          <User class="h-8 w-8 mx-auto mb-2 opacity-50" />
          <p class="text-sm">No data configured</p>
          <p class="text-xs mt-1">Configure panel display in the chatbot flow settings.</p>
        </div>

        <!-- Session Data with panel config -->
        <template v-else>
          <!-- Flow Name Badge -->
          <div v-if="sessionData.flow_name" class="flex items-center gap-2">
            <Badge variant="outline" class="text-xs">
              {{ sessionData.flow_name }}
            </Badge>
          </div>

          <!-- Dynamic Sections -->
          <div v-for="section in sortedSections" :key="section.id" class="space-y-2 border-t pt-4">
            <Collapsible
              v-if="section.collapsible"
              :open="!isSectionCollapsed(section.id)"
              @update:open="toggleSection(section.id)"
            >
              <CollapsibleTrigger class="flex items-center justify-between w-full py-2 text-sm font-medium hover:text-primary transition-colors">
                <span>{{ section.label }}</span>
                <ChevronDown
                  :class="[
                    'h-4 w-4 transition-transform',
                    isSectionCollapsed(section.id) ? '-rotate-90' : ''
                  ]"
                />
              </CollapsibleTrigger>
              <CollapsibleContent>
                <div
                  :class="[
                    'grid gap-2 pt-2',
                    section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1'
                  ]"
                >
                  <div
                    v-for="field in section.fields.sort((a, b) => a.order - b.order)"
                    :key="field.key"
                    class="bg-muted/50 rounded-md px-3 py-2"
                  >
                    <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">{{ field.label }}</p>
                    <!-- Badge display -->
                    <span
                      v-if="field.display_type === 'badge'"
                      :class="['inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1', getColorClass(field.color)]"
                    >
                      {{ getFieldValue(field.key) }}
                    </span>
                    <!-- Tag display -->
                    <span
                      v-else-if="field.display_type === 'tag'"
                      :class="['inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1', getColorClass(field.color)]"
                    >
                      {{ getFieldValue(field.key) }}
                    </span>
                    <!-- Default text display -->
                    <p v-else class="text-sm font-semibold break-words mt-0.5">{{ getFieldValue(field.key) }}</p>
                  </div>
                </div>
              </CollapsibleContent>
            </Collapsible>

            <!-- Non-collapsible section -->
            <div v-else>
              <h5 class="py-2 text-sm font-medium">{{ section.label }}</h5>
              <div
                :class="[
                  'grid gap-2',
                  section.columns === 2 ? 'grid-cols-2' : 'grid-cols-1'
                ]"
              >
                <div
                  v-for="field in section.fields.sort((a, b) => a.order - b.order)"
                  :key="field.key"
                  class="bg-muted/50 rounded-md px-3 py-2"
                >
                  <p class="text-[10px] uppercase tracking-wide text-muted-foreground font-medium">{{ field.label }}</p>
                  <!-- Badge display -->
                  <span
                    v-if="field.display_type === 'badge'"
                    :class="['inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold mt-1', getColorClass(field.color)]"
                  >
                    {{ getFieldValue(field.key) }}
                  </span>
                  <!-- Tag display -->
                  <span
                    v-else-if="field.display_type === 'tag'"
                    :class="['inline-flex items-center rounded-md px-2 py-1 text-xs font-medium mt-1', getColorClass(field.color)]"
                  >
                    {{ getFieldValue(field.key) }}
                  </span>
                  <!-- Default text display -->
                  <p v-else class="text-sm font-semibold break-words mt-0.5">{{ getFieldValue(field.key) }}</p>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Tags Section (always shown if tags exist) -->
        <div v-if="contactTags.length > 0" class="pt-4 border-t">
          <h5 class="py-2 text-sm font-medium">Tags</h5>
          <div class="flex flex-wrap gap-2">
            <Badge v-for="tag in contactTags" :key="tag" variant="secondary">
              {{ tag }}
            </Badge>
          </div>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>
