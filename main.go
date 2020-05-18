package main

import (
	"neo_explorer/core/config"
	"neo_explorer/core/log"
	"neo_explorer/neo/db"
	"neo_explorer/neo/rpc"
	"neo_explorer/neo/tasks"
)

func main() {
	log.Init()
	config.Load()
	db.Init()

	go rpc.TraceBestHeight()

	tasks.Run()

	select {}
}
