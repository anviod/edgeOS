<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Search, CornerDownLeft, SearchX } from 'lucide-vue-next'
import { navSections, type NavItem } from '@/lib/navigation'
import { useAlertStore } from '@/stores/alert'

const router = useRouter()
const alertStore = useAlertStore()

interface FlatEntry {
  item: NavItem
  sectionLabel: string
  showHeader: boolean
}

const open = ref(false)
const query = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const activeIndex = ref(0)
const listRef = ref<HTMLElement | null>(null)

const flatEntries = computed<FlatEntry[]>(() => {
  const q = query.value.trim().toLowerCase()
  const result: FlatEntry[] = []
  let lastSection = ''
  for (const section of navSections) {
    const collect = (items: NavItem[]) => {
      for (const item of items) {
        const matches = !q ||
          item.label.toLowerCase().includes(q) ||
          item.key.toLowerCase().includes(q) ||
          item.path.toLowerCase().includes(q)
        if (matches) {
          result.push({
            item,
            sectionLabel: section.label,
            showHeader: section.label !== lastSection,
          })
          lastSection = section.label
        }
        if (item.children) collect(item.children)
      }
    }
    collect(section.items)
  }
  return result
})

watch(open, async (val) => {
  if (val) {
    query.value = ''
    activeIndex.value = 0
    await nextTick()
    inputRef.value?.focus()
  }
})

watch(query, () => {
  activeIndex.value = 0
})

function openSearch() {
  open.value = true
}

function closeSearch() {
  open.value = false
}

function navigate(entry: FlatEntry) {
  open.value = false
  router.push(entry.item.path)
}

function onKeydown(e: KeyboardEvent) {
  if ((e.key === 'k' || e.key === 'K') && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    open.value ? closeSearch() : openSearch()
    return
  }
  if (!open.value) return
  if (e.key === 'Escape') {
    e.preventDefault()
    closeSearch()
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = Math.min(activeIndex.value + 1, flatEntries.value.length - 1)
    scrollActive()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = Math.max(activeIndex.value - 1, 0)
    scrollActive()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const entry = flatEntries.value[activeIndex.value]
    if (entry) navigate(entry)
  }
}

function scrollActive() {
  nextTick(() => {
    const el = listRef.value?.querySelector<HTMLElement>('[data-active="true"]')
    el?.scrollIntoView({ block: 'nearest' })
  })
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) closeSearch()
}

function rowKey(entry: FlatEntry, idx: number) {
  return `${entry.item.path}-${idx}`
}

defineExpose({ openSearch, closeSearch })

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-active-class="transition-opacity duration-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        class="gs-backdrop fixed inset-0 z-50 flex items-start justify-center p-4 pt-[12vh]"
        style="background: rgba(0, 0, 0, 0.5); backdrop-filter: blur(4px);"
        @click="handleOverlayClick"
      >
        <Transition
          enter-active-class="transition-all duration-200"
          enter-from-class="opacity-0 scale-95 -translate-y-2"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition-all duration-100"
          leave-from-class="opacity-100 scale-100 translate-y-0"
          leave-to-class="opacity-0 scale-95 -translate-y-2"
        >
          <div
            v-if="open"
            class="gs-panel w-full max-w-lg overflow-hidden rounded-xl"
            style="background: var(--bg-secondary); border: 1px solid var(--border-color); box-shadow: 0 24px 64px rgba(0,0,0,0.25);"
          >
            <!-- Input -->
            <div class="flex items-center gap-3 border-b px-4 py-3" style="border-color: var(--border-color);">
              <Search class="h-4 w-4 flex-shrink-0" style="color: var(--text-muted);" />
              <input
                ref="inputRef"
                v-model="query"
                type="text"
                placeholder="搜索页面、功能... (支持菜单名称 / 路径)"
                class="flex-1 bg-transparent text-sm outline-none"
                style="color: var(--text-primary);"
              />
              <kbd class="gs-kbd hidden text-xs sm:inline-block">Esc</kbd>
            </div>

            <!-- Results -->
            <div ref="listRef" class="custom-scrollbar max-h-[320px] overflow-y-auto py-1.5">
              <div v-if="flatEntries.length === 0" class="flex flex-col items-center gap-2 py-10">
                <SearchX class="h-8 w-8" style="color: var(--text-muted);" />
                <p class="text-sm" style="color: var(--text-muted);">未找到匹配项</p>
              </div>

              <template v-for="(entry, idx) in flatEntries" :key="rowKey(entry, idx)">
                <div
                  v-if="entry.showHeader"
                  class="px-4 pb-1 pt-2.5 text-[10px] font-semibold uppercase tracking-[0.16em]"
                  style="color: var(--text-muted);"
                >
                  {{ entry.sectionLabel }}
                </div>
                <button
                  type="button"
                  class="gs-item flex w-full items-center gap-3 px-4 py-2 text-left transition-colors"
                  :data-active="idx === activeIndex ? 'true' : 'false'"
                  :class="idx === activeIndex ? 'gs-item--active' : ''"
                  @mouseenter="activeIndex = idx"
                  @click="navigate(entry)"
                >
                  <span
                    class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg"
                    style="background: var(--bg-tertiary);"
                  >
                    <component
                      :is="entry.item.icon"
                      class="h-3.5 w-3.5"
                      :style="{ color: idx === activeIndex ? 'var(--accent-primary)' : 'var(--text-muted)' }"
                    />
                  </span>
                  <span class="flex-1 truncate text-sm" style="color: var(--text-primary);">{{ entry.item.label }}</span>
                  <span class="text-xs font-mono" style="color: var(--text-muted);">{{ entry.item.path }}</span>
                  <span
                    v-if="entry.item.badge === 'alerts' && alertStore.unacknowledgedCount > 0"
                    class="text-xs font-bold"
                    style="color: #EF4444;"
                  >
                    {{ alertStore.unacknowledgedCount }}
                  </span>
                  <CornerDownLeft v-if="idx === activeIndex" class="h-3.5 w-3.5" style="color: var(--text-muted);" />
                </button>
              </template>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.gs-kbd {
  color: var(--text-muted);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 1px 6px;
  font-family: monospace;
  background: var(--bg-tertiary);
}

.gs-item--active {
  background: var(--bg-hover);
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}
</style>
