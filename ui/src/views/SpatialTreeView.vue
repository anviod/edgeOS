<script setup lang="ts">
// 空间结构树视图 | Spatial structure tree view
// 支持两种根模式切换：
//   按节点（默认）: Node → Station → Building → Room → Device
//   按局站: Station → Building → Room → Device（跨节点汇聚，设备标记归属节点）
// | Spatial tree view with root mode switching:
//   node-rooted (default): Node → Station → Building → Room → Device
//   station-rooted: Station → Building → Room → Device (cross-node aggregation, device shows owning node)
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { RefreshCw, Server, Building2, Warehouse, DoorOpen, Cpu, ChevronRight, MapPin } from 'lucide-vue-next'
import { deviceApi } from '@/api/index'
import type { SpatialNode, SpatialStationRoot } from '@/types/edgeCore'
import StatusBadge from '@/components/edge/StatusBadge.vue'

const router = useRouter()

// ── 根模式 | root mode ──
const rootMode = ref<'node' | 'station'>('node')

// ── 数据 | data ──
const nodeTree = ref<SpatialNode[]>([])
const stationTree = ref<SpatialStationRoot[]>([])
const loading = ref(false)
const error = ref('')

// ── 展开状态 | expand/collapse state ──
const expandedNodes = ref<Set<string>>(new Set())
const expandedStations = ref<Set<string>>(new Set())
const expandedBuildings = ref<Set<string>>(new Set())
const expandedRooms = ref<Set<string>>(new Set())

// ── 汇总统计 | summary stats ──
const totalNodes = computed(() => nodeTree.value.length)
const totalStations = computed(() => {
  if (rootMode.value === 'node') {
    return nodeTree.value.reduce((sum, n) => sum + n.stations.length, 0)
  }
  return stationTree.value.length
})
const totalBuildings = computed(() => {
  if (rootMode.value === 'node') {
    return nodeTree.value.reduce((sum, n) =>
      sum + n.stations.reduce((s, st) => s + st.buildings.length, 0), 0)
  }
  return stationTree.value.reduce((sum, st) => sum + st.buildings.length, 0)
})
const totalDevices = computed(() => {
  if (rootMode.value === 'node') {
    return nodeTree.value.reduce((sum, n) => sum + n.device_count, 0)
  }
  return stationTree.value.reduce((sum, st) => sum + st.device_count, 0)
})

// ── key 构造 | key builders ──
function stationKey(prefix: string, stationCode: string) {
  return `${prefix}/${stationCode}`
}
function buildingKey(prefix: string, stationCode: string, buildingCode: string) {
  return `${prefix}/${stationCode}/${buildingCode}`
}
function roomKey(prefix: string, stationCode: string, buildingCode: string, roomCode: string) {
  return `${prefix}/${stationCode}/${buildingCode}/${roomCode}`
}

// ── toggle 函数 | toggle functions ──
function toggle(key: string, set: Set<string>) {
  if (set.has(key)) set.delete(key)
  else set.add(key)
}

function toggleNode(id: string) { toggle(id, expandedNodes.value) }
function toggleStation(prefix: string, code: string) { toggle(stationKey(prefix, code), expandedStations.value) }
function toggleBuilding(prefix: string, stCode: string, bdCode: string) { toggle(buildingKey(prefix, stCode, bdCode), expandedBuildings.value) }
function toggleRoom(prefix: string, stCode: string, bdCode: string, rmCode: string) { toggle(roomKey(prefix, stCode, bdCode, rmCode), expandedRooms.value) }

function isNodeExpanded(id: string) { return expandedNodes.value.has(id) }
function isStationExpanded(prefix: string, code: string) { return expandedStations.value.has(stationKey(prefix, code)) }
function isBuildingExpanded(prefix: string, stCode: string, bdCode: string) { return expandedBuildings.value.has(buildingKey(prefix, stCode, bdCode)) }
function isRoomExpanded(prefix: string, stCode: string, bdCode: string, rmCode: string) { return expandedRooms.value.has(roomKey(prefix, stCode, bdCode, rmCode)) }

// ── 全部展开/收起 | expand all / collapse all ──
function expandAll() {
  if (rootMode.value === 'node') {
    for (const n of nodeTree.value) {
      expandedNodes.value.add(n.node_id)
      for (const st of n.stations) {
        expandedStations.value.add(stationKey(n.node_id, st.station_code))
        for (const bd of st.buildings) {
          expandedBuildings.value.add(buildingKey(n.node_id, st.station_code, bd.building_code))
          for (const rm of bd.rooms) {
            expandedRooms.value.add(roomKey(n.node_id, st.station_code, bd.building_code, rm.room_code))
          }
        }
      }
    }
  } else {
    for (const st of stationTree.value) {
      expandedStations.value.add(stationKey('', st.station_code))
      for (const bd of st.buildings) {
        expandedBuildings.value.add(buildingKey('', st.station_code, bd.building_code))
        for (const rm of bd.rooms) {
          expandedRooms.value.add(roomKey('', st.station_code, bd.building_code, rm.room_code))
        }
      }
    }
  }
}

function collapseAll() {
  expandedNodes.value.clear()
  expandedStations.value.clear()
  expandedBuildings.value.clear()
  expandedRooms.value.clear()
}

// ── 加载数据 | load tree ──
async function loadTree() {
  loading.value = true
  error.value = ''
  try {
    if (rootMode.value === 'node') {
      const result = await deviceApi.getSpatialTree('node')
      nodeTree.value = Array.isArray(result) ? result : []
      // 默认展开第一层 | expand first level by default
      for (const n of nodeTree.value) {
        expandedNodes.value.add(n.node_id)
      }
    } else {
      const result = await deviceApi.getSpatialTree('station')
      stationTree.value = Array.isArray(result) ? result : []
      // 默认展开第一层 | expand first level by default
      for (const st of stationTree.value) {
        expandedStations.value.add(stationKey('', st.station_code))
      }
    }
  } catch (e) {
    error.value = (e as Error).message
    nodeTree.value = []
    stationTree.value = []
  } finally {
    loading.value = false
  }
}

// ── 切换根模式 | switch root mode ──
function switchMode(mode: 'node' | 'station') {
  if (rootMode.value === mode) return
  rootMode.value = mode
  // 清空展开状态 | clear expand state
  collapseAll()
  loadTree()
}

function navigateToDevice(nodeId: string, deviceId: string) {
  router.push(`/nodes/${nodeId}/devices/${deviceId}/points`)
}

// ── 监听模式变化自动加载 | auto-load on mode change ──
watch(rootMode, () => { /* switchMode already handles reload */ })

onMounted(loadTree)
</script>

<template>
  <div class="space-y-5">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold page-title">空间拓扑</h1>
        <p class="text-sm mt-1 page-subtitle">
          按 局站 → 机楼 → 机房 层级组织的设备空间结构树
        </p>
      </div>
      <div class="flex items-center gap-2">
        <!-- 根模式切换 | root mode toggle -->
        <div class="mode-toggle">
          <button
            :class="['mode-btn', rootMode === 'node' ? 'mode-btn--active' : '']"
            @click="switchMode('node')"
          >按节点</button>
          <button
            :class="['mode-btn', rootMode === 'station' ? 'mode-btn--active' : '']"
            @click="switchMode('station')"
          >按局站</button>
        </div>
        <button @click="expandAll" class="btn-ghost px-3 py-2 rounded-xl text-sm transition-colors">全部展开</button>
        <button @click="collapseAll" class="btn-ghost px-3 py-2 rounded-xl text-sm transition-colors">全部收起</button>
        <button @click="loadTree" class="btn-ghost flex items-center gap-2 px-4 py-2 rounded-xl text-sm transition-colors">
          <RefreshCw class="w-4 h-4" :class="loading ? 'animate-spin' : ''" style="width:16px;height:16px;" />
          刷新
        </button>
      </div>
    </div>

    <!-- Summary strip -->
    <div class="grid grid-cols-4 gap-4" v-if="totalDevices > 0 || totalNodes > 0">
      <div class="summary-card" v-if="rootMode === 'node'">
        <div class="summary-label">节点数</div>
        <div class="summary-value">{{ totalNodes }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">局站数</div>
        <div class="summary-value">{{ totalStations }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">机楼数</div>
        <div class="summary-value">{{ totalBuildings }}</div>
      </div>
      <div class="summary-card">
        <div class="summary-label">设备总数</div>
        <div class="summary-value">{{ totalDevices }}</div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading && totalDevices === 0 && totalNodes === 0" class="flex items-center justify-center py-16">
      <RefreshCw class="w-6 h-6 animate-spin" style="color: var(--accent-primary);" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-banner">{{ error }}</div>

    <!-- Empty -->
    <div v-else-if="totalDevices === 0 && totalNodes === 0" class="empty-state">
      <MapPin class="w-10 h-10 mb-3" style="color: var(--text-muted);" />
      <p class="text-base font-medium mb-1" style="color: var(--text-primary);">暂无空间数据</p>
      <p class="text-sm" style="color: var(--text-secondary);">等待 edgeCore 上报带空间属性的设备</p>
    </div>

    <!-- ==================== 节点根树 | Node-rooted tree ==================== -->
    <div v-else-if="rootMode === 'node'" class="tree-container">
      <template v-for="node in nodeTree" :key="node.node_id">
        <!-- L1: Node -->
        <div class="tree-node">
          <div class="tree-row tree-row--node" @click="toggleNode(node.node_id)">
            <ChevronRight class="tree-chevron" :class="isNodeExpanded(node.node_id) ? 'tree-chevron--open' : ''" />
            <Server class="tree-icon tree-icon--node" />
            <span class="tree-label tree-label--node">{{ node.node_name || node.node_id }}</span>
            <span class="tree-id font-mono">{{ node.node_id }}</span>
            <StatusBadge :status="node.status" size="sm" />
            <span class="tree-count-badge">{{ node.device_count }} 设备</span>
          </div>

          <!-- L2: Station -->
          <div v-show="isNodeExpanded(node.node_id)" class="tree-children">
            <template v-for="(station, si) in node.stations" :key="si">
              <div class="tree-row tree-row--station" style="paddingLeft: 2rem"
                @click="toggleStation(node.node_id, station.station_code)">
                <ChevronRight class="tree-chevron" :class="isStationExpanded(node.node_id, station.station_code) ? 'tree-chevron--open' : ''" />
                <Building2 class="tree-icon tree-icon--station" />
                <span class="tree-label">{{ station.station_name }}</span>
                <span v-if="station.station_code" class="tree-code font-mono">{{ station.station_code }}</span>
                <span class="tree-count-badge">{{ station.device_count }} 设备</span>
              </div>

              <!-- L3: Building -->
              <div v-show="isStationExpanded(node.node_id, station.station_code)" class="tree-children">
                <template v-for="(building, bi) in station.buildings" :key="bi">
                  <div class="tree-row tree-row--building" style="paddingLeft: 4rem"
                    @click="toggleBuilding(node.node_id, station.station_code, building.building_code)">
                    <ChevronRight class="tree-chevron" :class="isBuildingExpanded(node.node_id, station.station_code, building.building_code) ? 'tree-chevron--open' : ''" />
                    <Warehouse class="tree-icon tree-icon--building" />
                    <span class="tree-label">{{ building.building_name }}</span>
                    <span v-if="building.building_code" class="tree-code font-mono">{{ building.building_code }}</span>
                    <span class="tree-count-badge">{{ building.device_count }} 设备</span>
                  </div>

                  <!-- L4: Room -->
                  <div v-show="isBuildingExpanded(node.node_id, station.station_code, building.building_code)" class="tree-children">
                    <template v-for="(room, ri) in building.rooms" :key="ri">
                      <div class="tree-row tree-row--room" style="paddingLeft: 6rem"
                        @click="toggleRoom(node.node_id, station.station_code, building.building_code, room.room_code)">
                        <ChevronRight class="tree-chevron" :class="isRoomExpanded(node.node_id, station.station_code, building.building_code, room.room_code) ? 'tree-chevron--open' : ''" />
                        <DoorOpen class="tree-icon tree-icon--room" />
                        <span class="tree-label">{{ room.room_name }}</span>
                        <span v-if="room.room_code" class="tree-code font-mono">{{ room.room_code }}</span>
                        <span class="tree-count-badge">{{ room.device_count }} 设备</span>
                      </div>

                      <!-- L5: Device -->
                      <div v-show="isRoomExpanded(node.node_id, station.station_code, building.building_code, room.room_code)" class="tree-children">
                        <div v-for="device in room.devices" :key="device.device_id"
                          class="tree-row tree-row--device" style="paddingLeft: 8rem"
                          @click="navigateToDevice(node.node_id, device.device_id)">
                          <span class="tree-spacer"></span>
                          <Cpu class="tree-icon tree-icon--device" />
                          <span class="tree-label tree-label--device">{{ device.device_name || device.device_id }}</span>
                          <span class="tree-id font-mono">{{ device.device_id }}</span>
                          <StatusBadge :status="device.operating_state" size="sm" />
                          <span v-if="device.service_name" class="tree-meta">{{ device.service_name }}</span>
                        </div>
                      </div>
                    </template>
                  </div>
                </template>
              </div>
            </template>
          </div>
        </div>
      </template>
    </div>

    <!-- ==================== 局站根树 | Station-rooted tree ==================== -->
    <div v-else class="tree-container">
      <template v-for="(station, si) in stationTree" :key="si">
        <!-- L1: Station -->
        <div class="tree-node">
          <div class="tree-row tree-row--station-root" @click="toggleStation('', station.station_code)">
            <ChevronRight class="tree-chevron" :class="isStationExpanded('', station.station_code) ? 'tree-chevron--open' : ''" />
            <Building2 class="tree-icon tree-icon--station" />
            <span class="tree-label tree-label--node">{{ station.station_name }}</span>
            <span v-if="station.station_code" class="tree-code font-mono">{{ station.station_code }}</span>
            <span class="tree-count-badge">{{ station.device_count }} 设备</span>
          </div>

          <!-- L2: Building -->
          <div v-show="isStationExpanded('', station.station_code)" class="tree-children">
            <template v-for="(building, bi) in station.buildings" :key="bi">
              <div class="tree-row tree-row--building" style="paddingLeft: 2rem"
                @click="toggleBuilding('', station.station_code, building.building_code)">
                <ChevronRight class="tree-chevron" :class="isBuildingExpanded('', station.station_code, building.building_code) ? 'tree-chevron--open' : ''" />
                <Warehouse class="tree-icon tree-icon--building" />
                <span class="tree-label">{{ building.building_name }}</span>
                <span v-if="building.building_code" class="tree-code font-mono">{{ building.building_code }}</span>
                <span class="tree-count-badge">{{ building.device_count }} 设备</span>
              </div>

              <!-- L3: Room -->
              <div v-show="isBuildingExpanded('', station.station_code, building.building_code)" class="tree-children">
                <template v-for="(room, ri) in building.rooms" :key="ri">
                  <div class="tree-row tree-row--room" style="paddingLeft: 4rem"
                    @click="toggleRoom('', station.station_code, building.building_code, room.room_code)">
                    <ChevronRight class="tree-chevron" :class="isRoomExpanded('', station.station_code, building.building_code, room.room_code) ? 'tree-chevron--open' : ''" />
                    <DoorOpen class="tree-icon tree-icon--room" />
                    <span class="tree-label">{{ room.room_name }}</span>
                    <span v-if="room.room_code" class="tree-code font-mono">{{ room.room_code }}</span>
                    <span class="tree-count-badge">{{ room.device_count }} 设备</span>
                  </div>

                  <!-- L4: Device (with node badge) -->
                  <div v-show="isRoomExpanded('', station.station_code, building.building_code, room.room_code)" class="tree-children">
                    <div v-for="device in room.devices" :key="device.device_id"
                      class="tree-row tree-row--device" style="paddingLeft: 6rem"
                      @click="navigateToDevice(device.node_id || '', device.device_id)">
                      <span class="tree-spacer"></span>
                      <Cpu class="tree-icon tree-icon--device" />
                      <span class="tree-label tree-label--device">{{ device.device_name || device.device_id }}</span>
                      <span class="tree-id font-mono">{{ device.device_id }}</span>
                      <StatusBadge :status="device.operating_state" size="sm" />
                      <!-- 归属节点标记 | owning node badge -->
                      <span v-if="device.node_id" class="tree-node-badge">
                        <Server class="w-3 h-3" style="width:12px;height:12px;" />
                        {{ device.node_id }}
                      </span>
                      <span v-if="device.service_name" class="tree-meta">{{ device.service_name }}</span>
                    </div>
                  </div>
                </template>
              </div>
            </template>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.page-title { color: var(--text-primary); }
.page-subtitle { color: var(--text-secondary); }

/* ── 根模式切换 | root mode toggle ── */
.mode-toggle {
  display: flex;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 0.625rem;
  padding: 0.125rem;
  gap: 0.125rem;
}
.mode-btn {
  padding: 0.375rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-secondary);
  border-radius: 0.5rem;
  transition: all 0.15s ease;
  white-space: nowrap;
}
.mode-btn--active {
  background: var(--accent-primary);
  color: #fff;
}

.summary-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
}
.summary-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-bottom: 0.25rem;
}
.summary-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.error-banner {
  padding: 1rem 1.25rem;
  border-radius: 0.75rem;
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.2);
  color: #EF4444;
  font-size: 0.875rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 5rem 0;
  border-radius: 0.75rem;
  background: var(--bg-secondary);
  border: 1px dashed var(--border-color);
}

.tree-container {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  overflow: hidden;
}

.tree-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  cursor: pointer;
  transition: background-color 0.15s ease;
  user-select: none;
  border-bottom: 1px solid var(--border-color);
}
.tree-row:hover { background: var(--bg-hover); }
.tree-row:last-child { border-bottom: none; }

.tree-row--node { background: rgba(255, 255, 255, 0.015); }
.tree-row--station-root { background: rgba(255, 255, 255, 0.015); }
.tree-row--station { background: transparent; }
.tree-row--building { background: transparent; }
.tree-row--room { background: transparent; }
.tree-row--device { cursor: pointer; }
.tree-row--device:hover { background: rgba(14, 165, 233, 0.06); }

.tree-chevron {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  color: var(--text-muted);
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.tree-chevron--open { transform: rotate(90deg); }

.tree-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.tree-icon--node { color: var(--accent-primary); }
.tree-icon--station { color: #818CF8; }
.tree-icon--building { color: #A78BFA; }
.tree-icon--room { color: #C084FC; }
.tree-icon--device { color: var(--text-muted); }

.tree-label {
  font-size: 0.875rem;
  color: var(--text-primary);
  font-weight: 500;
}
.tree-label--node { font-weight: 600; }
.tree-label--device { font-weight: 400; color: var(--text-secondary); }

.tree-id {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.tree-code {
  font-size: 0.75rem;
  color: var(--text-muted);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  background: rgba(99, 102, 241, 0.08);
}

.tree-count-badge {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--text-secondary);
  padding: 0.125rem 0.5rem;
  border-radius: 0.375rem;
  background: rgba(255, 255, 255, 0.04);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

.tree-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
  flex-shrink: 0;
}

.tree-node-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.6875rem;
  color: var(--text-muted);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  background: rgba(14, 165, 233, 0.06);
  border: 1px solid rgba(14, 165, 233, 0.12);
  flex-shrink: 0;
  font-family: monospace;
}

.tree-spacer {
  width: 16px;
  flex-shrink: 0;
}

.tree-children {
  /* 子节点容器——无额外样式，仅用于 v-show 控制 */
}
</style>
