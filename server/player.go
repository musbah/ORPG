package main

import (
	"sync"
)

//TODO: remove this, just temporary to test multiple players at the same time
var idMutex sync.Mutex
var id uint32

type player struct {
	id       uint32
	mapIndex uint32
	x        uint32
	y        uint32
}

//TODO: actually load player information later on
func loadPlayer() *player {

	idMutex.Lock()
	player := player{id: id}
	id++
	idMutex.Unlock()

	return &player
}
