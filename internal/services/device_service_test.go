package services

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anviod/edgeOS/internal/model"
)

// ======================== DeviceService ========================

func TestDeviceService_UpsertAndGet(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	dev := &model.EdgeCoreDeviceInfo{
		DeviceID:   "dev-1",
		DeviceName: "Sensor A",
		AdminState: "UNLOCKED",
	}
	require.NoError(t, svc.UpsertDevice("node-1", dev))

	got, err := svc.GetDevice("node-1", "dev-1")
	require.NoError(t, err)
	assert.Equal(t, "dev-1", got.DeviceID)
	assert.Equal(t, "Sensor A", got.DeviceName)
	assert.Greater(t, got.LastSync, int64(0))
}

func TestDeviceService_GetDevice_NotFound(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	_, err := svc.GetDevice("node-x", "dev-x")
	assert.Error(t, err)
}

func TestDeviceService_ListDevices(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// 空节点
	devs, err := svc.ListDevices("node-1")
	require.NoError(t, err)
	assert.Empty(t, devs)

	// 写入 node-1 的 2 个设备和 node-2 的 1 个设备
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeCoreDeviceInfo{DeviceID: "d1"}))
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeCoreDeviceInfo{DeviceID: "d2"}))
	require.NoError(t, svc.UpsertDevice("node-2", &model.EdgeCoreDeviceInfo{DeviceID: "d3"}))

	devs, err = svc.ListDevices("node-1")
	require.NoError(t, err)
	assert.Len(t, devs, 2)

	devs2, err := svc.ListDevices("node-2")
	require.NoError(t, err)
	assert.Len(t, devs2, 1)
}

func TestDeviceService_DeleteDevice(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeCoreDeviceInfo{DeviceID: "del-dev"}))

	require.NoError(t, svc.DeleteDevice("node-1", "del-dev"))

	_, err := svc.GetDevice("node-1", "del-dev")
	assert.Error(t, err)
}

func TestDeviceService_CountDevices(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	assert.Equal(t, 0, svc.CountDevices())

	for i := 0; i < 5; i++ {
		require.NoError(t, svc.UpsertDevice("node-1", &model.EdgeCoreDeviceInfo{
			DeviceID: fmt.Sprintf("d%d", i),
		}))
	}
	assert.Equal(t, 5, svc.CountDevices())
}

func TestDeviceService_ReconcileDevices_PrunesMissing(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "old-a", DeviceName: "A"}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "old-b", DeviceName: "B"}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "keep", DeviceName: "Keep"}))
	require.NoError(t, svc.UpsertDevice("n2", &model.EdgeCoreDeviceInfo{DeviceID: "other", DeviceName: "Other"}))

	upserted, removed, err := svc.ReconcileDevices("n1", []model.EdgeCoreDeviceInfo{
		{DeviceID: "keep", DeviceName: "Keep Updated"},
		{DeviceID: "new-c", DeviceName: "C"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, upserted)
	assert.Equal(t, 2, removed)

	devs, err := svc.ListDevices("n1")
	require.NoError(t, err)
	require.Len(t, devs, 2)
	ids := map[string]string{}
	for _, d := range devs {
		ids[d.DeviceID] = d.DeviceName
	}
	assert.Equal(t, "Keep Updated", ids["keep"])
	assert.Equal(t, "C", ids["new-c"])

	// 其他节点不受影响
	other, err := svc.GetDevice("n2", "other")
	require.NoError(t, err)
	assert.Equal(t, "Other", other.DeviceName)
	assert.Equal(t, 3, svc.CountDevices())
}

func TestDeviceService_ReconcileDevices_EmptyClearsNode(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "gone"}))

	upserted, removed, err := svc.ReconcileDevices("n1", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, upserted)
	assert.Equal(t, 1, removed)
	devs, err := svc.ListDevices("n1")
	require.NoError(t, err)
	assert.Empty(t, devs)
}

// ======================== 空间属性 | Spatial attributes ========================

func TestDeviceService_SpatialAttributes_PersistAndRetrieve(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	dev := &model.EdgeCoreDeviceInfo{
		DeviceID:    "spatial-1",
		DeviceName:  "温度传感器A",
		StationName: "海府一体化冷站",
		StationCode: "HKO.HFJLZ",
		RoomName:    "海府动力机房/1楼/1号电力室",
		RoomCode:    "HKO.HFJDD01",
	}
	require.NoError(t, svc.UpsertDevice("n1", dev))

	got, err := svc.GetDevice("n1", "spatial-1")
	require.NoError(t, err)
	assert.Equal(t, "海府一体化冷站", got.StationName)
	assert.Equal(t, "HKO.HFJLZ", got.StationCode)
	assert.Equal(t, "海府动力机房/1楼/1号电力室", got.RoomName)
	assert.Equal(t, "HKO.HFJDD01", got.RoomCode)
}

func TestDeviceService_SpatialAttributes_OmitemptyBackwardCompat(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// 不设空间属性的设备——确保 omitempty 不影响序列化/反序列化
	dev := &model.EdgeCoreDeviceInfo{
		DeviceID:   "legacy-1",
		DeviceName: "Legacy Device",
	}
	require.NoError(t, svc.UpsertDevice("n1", dev))

	got, err := svc.GetDevice("n1", "legacy-1")
	require.NoError(t, err)
	assert.Equal(t, "", got.StationName)
	assert.Equal(t, "", got.StationCode)
	assert.Equal(t, "", got.RoomName)
	assert.Equal(t, "", got.RoomCode)
}

func TestDeviceService_ReconcileDevices_PreservesSpatialAttributes(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// 首次写入带空间属性的设备
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID:    "dev-spatial",
		DeviceName:  "旧名称",
		StationName: "局站A",
		StationCode: "ST.A",
		RoomName:    "机房A",
		RoomCode:    "RM.A",
	}))

	// Reconcile 更新该设备——空间属性应随上报数据更新
	_, _, err := svc.ReconcileDevices("n1", []model.EdgeCoreDeviceInfo{
		{
			DeviceID:    "dev-spatial",
			DeviceName:  "新名称",
			StationName: "局站B",
			StationCode: "ST.B",
			RoomName:    "机房B",
			RoomCode:    "RM.B",
		},
	})
	require.NoError(t, err)

	got, err := svc.GetDevice("n1", "dev-spatial")
	require.NoError(t, err)
	assert.Equal(t, "新名称", got.DeviceName)
	assert.Equal(t, "局站B", got.StationName)
	assert.Equal(t, "ST.B", got.StationCode)
	assert.Equal(t, "机房B", got.RoomName)
	assert.Equal(t, "RM.B", got.RoomCode)
}

// ======================== 空间树 | Spatial tree ========================

func TestDeviceService_ListAllDevices(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "d1"}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{DeviceID: "d2"}))
	require.NoError(t, svc.UpsertDevice("n2", &model.EdgeCoreDeviceInfo{DeviceID: "d3"}))

	all, err := svc.ListAllDevices()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// 验证 nodeID 正确解析
	nodeIDs := map[string]bool{}
	for _, nd := range all {
		nodeIDs[nd.NodeID] = true
	}
	assert.True(t, nodeIDs["n1"])
	assert.True(t, nodeIDs["n2"])
}

func TestDeviceService_BuildSpatialTree_HierarchyAndUnassigned(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// n1 下有 3 个设备：
	//   - dev1: 局站A / 机楼B1 / 机房A1
	//   - dev2: 局站A / 机楼B1 / 机房A2
	//   - dev3: 无空间属性
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "dev1", DeviceName: "设备1",
		StationName: "局站A", StationCode: "ST.A",
		BuildingName: "机楼B1", BuildingCode: "BD.B1",
		RoomName: "机房A1", RoomCode: "RM.A1",
	}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "dev2", DeviceName: "设备2",
		StationName: "局站A", StationCode: "ST.A",
		BuildingName: "机楼B1", BuildingCode: "BD.B1",
		RoomName: "机房A2", RoomCode: "RM.A2",
	}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "dev3", DeviceName: "设备3（无空间）",
	}))

	nodes := []NodeBrief{
		{NodeID: "n1", NodeName: "Node 1", Status: "online"},
	}
	tree, err := svc.BuildSpatialTree(nodes)
	require.NoError(t, err)
	require.Len(t, tree, 1)

	node := tree[0]
	assert.Equal(t, "n1", node.NodeID)
	assert.Equal(t, "Node 1", node.NodeName)
	assert.Equal(t, 3, node.DeviceCount)

	// 应有 2 个局站：局站A + 未分配
	require.Len(t, node.Stations, 2)

	// 第一个局站：局站A，含 2 个设备、1 个机楼
	stA := node.Stations[0]
	assert.Equal(t, "局站A", stA.StationName)
	assert.Equal(t, "ST.A", stA.StationCode)
	assert.Equal(t, 2, stA.DeviceCount)
	require.Len(t, stA.Buildings, 1)
	assert.Equal(t, "机楼B1", stA.Buildings[0].BuildingName)
	assert.Equal(t, "BD.B1", stA.Buildings[0].BuildingCode)
	// 机楼下有 2 个机房
	require.Len(t, stA.Buildings[0].Rooms, 2)
	assert.Equal(t, "机房A1", stA.Buildings[0].Rooms[0].RoomName)
	assert.Equal(t, "RM.A1", stA.Buildings[0].Rooms[0].RoomCode)
	assert.Len(t, stA.Buildings[0].Rooms[0].Devices, 1)
	assert.Equal(t, "机房A2", stA.Buildings[0].Rooms[1].RoomName)
	assert.Len(t, stA.Buildings[0].Rooms[1].Devices, 1)

	// 第二个局站：未分配，含 1 个设备、1 个机楼（未分配）
	stU := node.Stations[1]
	assert.Equal(t, "未分配", stU.StationName)
	assert.Equal(t, "", stU.StationCode)
	assert.Equal(t, 1, stU.DeviceCount)
	require.Len(t, stU.Buildings, 1)
	assert.Equal(t, "未分配", stU.Buildings[0].BuildingName)
	require.Len(t, stU.Buildings[0].Rooms, 1)
	assert.Equal(t, "未分配", stU.Buildings[0].Rooms[0].RoomName)
	assert.Len(t, stU.Buildings[0].Rooms[0].Devices, 1)
	assert.Equal(t, "dev3", stU.Buildings[0].Rooms[0].Devices[0].DeviceID)
}

func TestDeviceService_BuildSpatialTree_EmptyNode(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)

	// 无设备的节点——应返回空 stations 列表
	tree, err := svc.BuildSpatialTree([]NodeBrief{
		{NodeID: "empty-node", NodeName: "Empty", Status: "online"},
	})
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "empty-node", tree[0].NodeID)
	assert.Equal(t, 0, tree[0].DeviceCount)
	assert.Empty(t, tree[0].Stations)
}

func TestDeviceService_ListByStationCode(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "d1", StationCode: "ST.A",
	}))
	require.NoError(t, svc.UpsertDevice("n2", &model.EdgeCoreDeviceInfo{
		DeviceID: "d2", StationCode: "ST.A",
	}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "d3", StationCode: "ST.B",
	}))

	results, err := svc.ListByStationCode("ST.A")
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// 不存在的局站
	empty, err := svc.ListByStationCode("ST.NOTEXIST")
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestDeviceService_ListByRoomCode(t *testing.T) {
	db, cleanup := openTestDB(t)
	defer cleanup()

	svc := NewDeviceService(db)
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "d1", RoomCode: "RM.X",
	}))
	require.NoError(t, svc.UpsertDevice("n2", &model.EdgeCoreDeviceInfo{
		DeviceID: "d2", RoomCode: "RM.X",
	}))
	require.NoError(t, svc.UpsertDevice("n1", &model.EdgeCoreDeviceInfo{
		DeviceID: "d3", RoomCode: "RM.Y",
	}))

	results, err := svc.ListByRoomCode("RM.X")
	require.NoError(t, err)
	assert.Len(t, results, 2)
}
