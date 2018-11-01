package common

import "time"

// Constants
const (
	KeyTick       = 50 * time.Millisecond
	TotalGameMaps = 1

	//TODO: change capacity depending on max X and max Y
	MaxIntToBytesLength = 4

	MovementByte    = byte('M')
	OtherPlayerByte = byte('P')
)
