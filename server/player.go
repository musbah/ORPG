package main

type player struct {
	x uint32
	y uint32
}

//TODO: actually load player information later on
func loadPlayer() *player {
	player := player{}
	return &player
}
