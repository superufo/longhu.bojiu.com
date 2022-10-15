package server

import (
	"github.com/shopspring/decimal"
	"math/rand"
)

/*
** 赔率
**
 */
func do_new_desk_bet(roomId int32, deskId int32) {
	pos0 := decimal.New(rand.Int63n(900000000), 0)
	pos01 := decimal.New(int64(100000000), 0)
	pos02 := decimal.New(int64(10000), 0)
	pos0 = pos0.Sub(pos01).Div(pos02).Mul(pos02)

	pos1 := decimal.New(rand.Int63n(5000000), 0)
	pos11 := decimal.New(int64(10000000), 0)
	pos1 = pos0.Sub(pos11).Div(pos02).Mul(pos02)

	pos2 := decimal.New(rand.Int63n(900000000), 0)
	pos21 := decimal.New(int64(100000000), 0)
	pos2 = pos0.Sub(pos21).Div(pos02).Mul(pos02)
	pso := map[int32]int64{
		0: pos0.IntPart(),
		1: pos1.IntPart(),
		2: pos2.IntPart(),
	}
	for k, _ := range POS_PRICE {
		LongHuGameRoom.Rooms[roomId].Desks[deskId].DeskBet[k] = pso[k]
	}
}
