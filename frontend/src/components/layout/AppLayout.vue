<script setup lang="ts">
import { ref, computed } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Separator } from '@/components/ui/separator'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from '@/components/ui/popover'
import {
  LayoutDashboard,
  MessageSquare,
  Bot,
  FileText,
  Megaphone,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Users,
  Workflow,
  Sparkles,
  Key,
  User,
  UserX
} from 'lucide-vue-next'
import { getInitials } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const isCollapsed = ref(false)
const isUserMenuOpen = ref(false)

// Define all navigation items with role requirements
const allNavItems = [
  {
    name: 'Dashboard',
    path: '/',
    icon: LayoutDashboard,
    roles: ['admin', 'manager']
  },
  {
    name: 'Chat',
    path: '/chat',
    icon: MessageSquare,
    roles: ['admin', 'manager', 'agent']
  },
  {
    name: 'Chatbot',
    path: '/chatbot',
    icon: Bot,
    roles: ['admin', 'manager'],
    children: [
      { name: 'Overview', path: '/chatbot', icon: Bot },
      { name: 'Settings', path: '/chatbot/settings', icon: Settings },
      { name: 'Keywords', path: '/chatbot/keywords', icon: Key },
      { name: 'Flows', path: '/chatbot/flows', icon: Workflow },
      { name: 'AI Contexts', path: '/chatbot/ai', icon: Sparkles },
      { name: 'Transfers', path: '/chatbot/transfers', icon: UserX }
    ]
  },
  {
    name: 'Transfers',
    path: '/chatbot/transfers',
    icon: UserX,
    roles: ['agent']
  },
  {
    name: 'Templates',
    path: '/templates',
    icon: FileText,
    roles: ['admin', 'manager']
  },
  {
    name: 'Flows',
    path: '/flows',
    icon: Workflow,
    roles: ['admin', 'manager']
  },
  {
    name: 'Campaigns',
    path: '/campaigns',
    icon: Megaphone,
    roles: ['admin', 'manager']
  },
  {
    name: 'Settings',
    path: '/settings',
    icon: Settings,
    roles: ['admin', 'manager'],
    children: [
      { name: 'General', path: '/settings', icon: Settings },
      { name: 'Accounts', path: '/settings/accounts', icon: Users },
      { name: 'Users', path: '/settings/users', icon: Users, roles: ['admin'] },
      { name: 'API Keys', path: '/settings/api-keys', icon: Key, roles: ['admin'] }
    ]
  }
]

// Filter navigation based on user role
const navigation = computed(() => {
  const userRole = authStore.userRole || 'agent'

  return allNavItems
    .filter(item => item.roles.includes(userRole))
    .map(item => ({
      ...item,
      active: item.path === '/'
        ? route.name === 'dashboard'
        : item.path === '/chat'
          ? route.name === 'chat' || route.name === 'chat-conversation'
          : route.path.startsWith(item.path),
      children: item.children?.filter(
        child => !child.roles || child.roles.includes(userRole)
      )
    }))
})

const toggleSidebar = () => {
  isCollapsed.value = !isCollapsed.value
}

const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-screen bg-background">
    <!-- Sidebar -->
    <aside
      :class="[
        'flex flex-col border-r bg-card transition-all duration-300',
        isCollapsed ? 'w-16' : 'w-64'
      ]"
    >
      <!-- Logo -->
      <div class="flex h-16 items-center justify-between px-4 border-b">
        <RouterLink to="/" class="flex items-center gap-2">
          <div class="h-8 w-8 rounded-lg bg-primary flex items-center justify-center">
            <MessageSquare class="h-5 w-5 text-primary-foreground" />
          </div>
          <span
            v-if="!isCollapsed"
            class="font-bold text-lg text-foreground"
          >
            Whatomate
          </span>
        </RouterLink>
        <Button
          variant="ghost"
          size="icon"
          class="h-8 w-8"
          @click="toggleSidebar"
        >
          <ChevronLeft v-if="!isCollapsed" class="h-4 w-4" />
          <ChevronRight v-else class="h-4 w-4" />
        </Button>
      </div>

      <!-- Navigation -->
      <ScrollArea class="flex-1 py-4">
        <nav class="space-y-1 px-2">
          <template v-for="item in navigation" :key="item.path">
            <RouterLink
              :to="item.path"
              :class="[
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                item.active
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                isCollapsed && 'justify-center px-2'
              ]"
            >
              <component :is="item.icon" class="h-5 w-5 shrink-0" />
              <span v-if="!isCollapsed">{{ item.name }}</span>
            </RouterLink>

            <!-- Submenu items -->
            <template v-if="item.children && item.active && !isCollapsed">
              <RouterLink
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                :class="[
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ml-4',
                  route.path === child.path
                    ? 'bg-primary/10 text-primary'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                ]"
              >
                <component :is="child.icon" class="h-4 w-4 shrink-0" />
                <span>{{ child.name }}</span>
              </RouterLink>
            </template>
          </template>
        </nav>
      </ScrollArea>

      <!-- User section -->
      <div class="border-t p-4">
        <Popover v-model:open="isUserMenuOpen">
          <PopoverTrigger as-child>
            <Button
              variant="ghost"
              :class="[
                'flex items-center w-full h-auto px-2 py-2 gap-3',
                isCollapsed && 'justify-center'
              ]"
            >
              <Avatar class="h-8 w-8">
                <AvatarImage :src="undefined" />
                <AvatarFallback>
                  {{ getInitials(authStore.user?.full_name || 'U') }}
                </AvatarFallback>
              </Avatar>
              <div v-if="!isCollapsed" class="flex flex-col items-start text-left">
                <span class="text-sm font-medium truncate max-w-[140px]">
                  {{ authStore.user?.full_name }}
                </span>
                <span class="text-xs text-muted-foreground truncate max-w-[140px]">
                  {{ authStore.user?.email }}
                </span>
              </div>
            </Button>
          </PopoverTrigger>
          <PopoverContent side="top" align="start" class="w-56 p-2">
            <div class="text-sm font-medium px-2 py-1.5 text-muted-foreground">My Account</div>
            <Separator class="my-1" />
            <RouterLink to="/profile">
              <Button
                variant="ghost"
                class="w-full justify-start px-2 py-1.5 h-auto font-normal"
                @click="isUserMenuOpen = false"
              >
                <User class="mr-2 h-4 w-4" />
                <span>Profile</span>
              </Button>
            </RouterLink>
            <Button
              variant="ghost"
              class="w-full justify-start px-2 py-1.5 h-auto font-normal"
              @click="handleLogout"
            >
              <LogOut class="mr-2 h-4 w-4" />
              <span>Log out</span>
            </Button>
          </PopoverContent>
        </Popover>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 overflow-hidden">
      <RouterView />
    </main>
  </div>
</template>
