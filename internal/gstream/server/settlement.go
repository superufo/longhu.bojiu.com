package server

import (
	"github.com/shopspring/decimal"
	"longhu.bojiu.com/enum"
	cproto "longhu.bojiu.com/internal/gstream/proto"
	"math/rand"
	"time"
)

// 更具库存生成当前局的结果
func make_result(deskId int32, RoomId int32, GamesInfo *cproto.StorageInfo, GamesConfig *cproto.StorageConfig) (int32, int32, int32, int32) {
	var StockState int32 = 0
	do_make_stock_result(GamesInfo, GamesConfig, &StockState)
	switch StockState {
	case 1:
		ishe, win := make_result_1(deskId, RoomId)
		for {
			re, long, hu := make_result_2()
			if re == 1 && ishe == true {
				return StockState, re, long, hu
			} else if re != 1 && win == re {
				return StockState, re, long, hu
			}
		}
	default:
		re, long, hu := make_result_2()
		return StockState, re, long, hu
	}

}
func do_make_stock_result(GamesInfo *cproto.StorageInfo, GamesConfig *cproto.StorageConfig, StockState *int32) int32 {
	if GamesInfo.CurrentStock1 <= GamesConfig.Stock1WarnWater {
		*StockState = 1
		return 1
	} else {
		if GamesInfo.CurrentStock1 >= GamesConfig.Stock1 {
			if GamesInfo.CurrentStock1 > GamesConfig.Stock2WarnWater {
				*StockState = 2
				return 2
			} else {
				*StockState = 0
				return 0
			}
		} else {
			randdata := rand.Int63n(10000)
			if randdata < GamesConfig.DrawWater {
				*StockState = 1
				return 1
			} else {
				*StockState = 0
				return 0
			}
		}
	}
}

func make_result_1(deskId int32, RoomId int32) (bool, int32) {
	AllPlayerBet := LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerBet
	//map[string]map[int]int64
	var allbet int64
	var ishe bool
	kvs := map[int32]int64{0: 0, 1: 0, 2: 0}
	for _, PlayerBet := range AllPlayerBet {
		for k, v := range PlayerBet {
			kvs[k] += v
			allbet += v
		}
	}

	Num1 := decimal.New(kvs[1], 0)
	Num2 := decimal.NewFromFloat32(POS_PRICE[1])
	Num3 := Num1.Mul(Num2).IntPart()

	if Num3 >= allbet {
		ishe = false
	} else {
		ishe = false
	}
	if kvs[0] > kvs[2] {
		return ishe, int32(2)
	} else {
		return ishe, int32(0)
	}

}

var Cards = []int32{101, 201, 301, 401, 501, 601, 701, 801, 901, 1001, 1101, 1201, 1301, // 黑桃
	102, 202, 302, 402, 502, 602, 702, 802, 902, 1002, 1102, 1202, 1302, // 红桃
	103, 203, 303, 403, 503, 603, 703, 803, 903, 1003, 1103, 1203, 1303, // 梅花
	104, 204, 304, 404, 504, 604, 704, 804, 904, 1004, 1104, 1204, 1304} // 方块

func make_result_2() (int32, int32, int32) {
	tmpSlice := make([]int32, len(Cards))
	copy(tmpSlice, Cards)

	// 打乱顺序
	rand.Seed(time.Now().Unix())
	rand.Shuffle(len(tmpSlice), func(i int, j int) {
		tmpSlice[i], tmpSlice[j] = tmpSlice[j], tmpSlice[i]
	})
	var long int32 = -1
	var hu int32 = -1
	var re int32 = 1
	for k, v := range tmpSlice {
		if long == -1 {
			long = v
			tmpSlice[k] = 0
		}
	}
	for _, v := range tmpSlice {
		if v != 0 && hu == -1 {
			hu = v
		}
	}
	if long == hu {
		re = 1
	} else if long > hu {
		re = 0
	} else {
		re = 2
	}
	return re, long, hu
}

func settlement(SidState []*cproto.StorageCtrlUserCtrl, deskId int32, RoomId int32, StockState *int32, newresult int32, GamesInfo *cproto.StorageInfo, GamesConfig *cproto.StorageConfig) map[string]enum.TmpSettlement {
	var TmpSettlement = make(map[string]enum.TmpSettlement)
	for _, vSidState := range SidState {
		Sid, oneSettlement := settlement_1(vSidState, deskId, RoomId, StockState, newresult, GamesInfo, GamesConfig)
		TmpSettlement[Sid] = oneSettlement
	}
	return TmpSettlement
}
func settlement_1(vSidState *cproto.StorageCtrlUserCtrl, deskId int32, RoomId int32, StockState *int32, newresult int32, GamesInfo *cproto.StorageInfo, GamesConfig *cproto.StorageConfig) (string, enum.TmpSettlement) {
	onePlayerBet := LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerBet[vSidState.SId]
	var allBets int64 = 0
	for _, v := range onePlayerBet {
		allBets += v
	}
	if _, ok := onePlayerBet[newresult]; ok {
		UserBet := decimal.New(onePlayerBet[newresult], 0)
		PriceData := decimal.NewFromFloat32(POS_PRICE[newresult])
		PlayerServiceCharge := decimal.NewFromFloat32(GamesConfig.PlayerServiceCharge)
		RealPlayerServiceCharge := UserBet.Mul(PriceData).Sub(UserBet).Mul(PlayerServiceCharge)
		addGold := UserBet.Mul(PriceData).Sub(RealPlayerServiceCharge)
		win := addGold.IntPart() - allBets
		var TmpSettlement enum.TmpSettlement
		TmpSettlement.Win = win
		TmpSettlement.AddGold = addGold.IntPart()
		TmpSettlement.AllBets = allBets
		TmpSettlement.PlayerServiceCharge = PlayerServiceCharge.IntPart()
		TmpSettlement.StockState = *StockState
		make_new_state_for_game_info_config(GamesConfig, GamesInfo, win, StockState)

		return vSidState.SId, TmpSettlement
	} else {
		win := -allBets
		make_new_state_for_game_info_config(GamesConfig, GamesInfo, win, StockState)
		var TmpSettlement enum.TmpSettlement
		TmpSettlement.Win = win
		TmpSettlement.AddGold = 0
		TmpSettlement.AllBets = allBets
		TmpSettlement.PlayerServiceCharge = 0
		return vSidState.SId, TmpSettlement
	}
}
func make_new_state_for_game_info_config(GamesConfig *cproto.StorageConfig, GamesInfo *cproto.StorageInfo, Win int64, StockState *int32) {
	if Win == 0 {
		do_make_stock_result(GamesInfo, GamesConfig, StockState)
	} else if Win > 0 {
		CurrentStock1 := GamesInfo.CurrentStock1 - Win
		switch *StockState {
		case 2:
			CurrentStock2 := GamesInfo.CurrentStock2 - Win
			GamesInfo.CurrentStock2 = CurrentStock2
			GamesInfo.CurrentStock1 = CurrentStock1
			do_make_stock_result(GamesInfo, GamesConfig, StockState)
		default:
			GamesInfo.CurrentStock1 = CurrentStock1
			do_make_stock_result(GamesInfo, GamesConfig, StockState)
		}
	} else {
		tmpWin := decimal.New(Win, 0)
		Stock2ServiceCharge := decimal.NewFromFloat32(GamesConfig.Stock2ServiceCharge)
		ToStock1 := decimal.NewFromFloat32(GamesConfig.ToStock1)
		num2 := tmpWin.Mul(Stock2ServiceCharge).IntPart()
		num1 := tmpWin.Mul(ToStock1).IntPart()
		CurrentStock1 := GamesInfo.CurrentStock1 - num1
		CurrentStock2 := GamesInfo.CurrentStock2 - num2
		GamesInfo.CurrentStock2 = CurrentStock2
		GamesInfo.CurrentStock1 = CurrentStock1
		do_make_stock_result(GamesInfo, GamesConfig, StockState)
	}
}
