package main

import (
	"encoding/json"
	"fmt"
	"os"

	"go.etcd.io/bbolt"

	"github.com/anviod/edgeOS/internal/model"
	"github.com/anviod/edgeOS/internal/services"
)

// 一次性/运维对账：按全量设备快照剪枝指定节点。
// 用法:
//
//	go run ./cmd/reconcile-devices -- <dbPath> <nodeID> <devices.json>
//
// devices.json: [{"device_id":"...","device_name":"..."}, ...]
func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "usage: reconcile-devices <dbPath> <nodeID> <devices.json>\n")
		os.Exit(2)
	}
	dbPath, nodeID, devicesFile := os.Args[1], os.Args[2], os.Args[3]

	raw, err := os.ReadFile(devicesFile)
	if err != nil {
		panic(err)
	}
	var reported []model.EdgeXDeviceInfo
	if err := json.Unmarshal(raw, &reported); err != nil {
		panic(err)
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	svc := services.NewDeviceService(db)
	reg := services.NewRegistryService(db)
	before := svc.CountDevices()
	up, rm, err := svc.ReconcileDevices(nodeID, reported)
	if err != nil {
		panic(err)
	}
	if err := reg.EnsureNodeOnline(nodeID, nodeID, "ean"); err != nil {
		panic(err)
	}
	after := svc.CountDevices()
	devs, _ := svc.ListDevices(nodeID)
	fmt.Printf("before=%d upserted=%d removed=%d after=%d\n", before, up, rm, after)
	for _, d := range devs {
		fmt.Printf(" - %s (%s)\n", d.DeviceID, d.DeviceName)
	}
}
