package main

//TODO: remove this, just temporary to test multiple players at the same time
var id uint32

type player struct {
	id       uint32
	mapIndex uint32
	x        uint32
	y        uint32
}

//TODO: actually load player information later on
func loadPlayer() *player {
	player := player{id: id}
	id++
	return &player
}
