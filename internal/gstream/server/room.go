package server

import (
	"bytes"
	db_log "common.bojiu.com/models/bj_log"
	"context"
	"encoding/json"
	"fmt"
	toolProto "github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"longhu.bojiu.com/enum"
	"longhu.bojiu.com/internal/gstream/pb"
	cproto "longhu.bojiu.com/internal/gstream/proto"
	protoStruct "longhu.bojiu.com/internal/proto"
	"longhu.bojiu.com/pkg/log"
	"longhu.bojiu.com/pkg/mysql"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

var LongHuGameRoom *enum.LHGameRoom
var DeskId *enum.LHDeskId
var POS_PRICE map[int32]float32
var DeskInit map[int32]map[int32][]int32

var SSLock *sync.RWMutex

func StartGame() {
	creatRoom()
	InitData()
	go AddNewResult()

	// 启动定时器 唯一启动状态是waitTimer
}
func creatRoom() {
	data := &enum.RoomData{
		State:  1,
		MaxBet: 0,
		MinBet: 0,
		Desks:  make(map[int32]*enum.DeskInfo),
	}
	LongHuGameRoom = &enum.LHGameRoom{}
	LongHuGameRoom.Rooms = make(map[int32]*enum.RoomData)
	LongHuGameRoom.Rooms[enum.PM] = data
	LongHuGameRoom.SSLock = new(sync.RWMutex)
	fmt.Println("创建的场次信息----", &LongHuGameRoom.Rooms)
}

func InitData() {

	DeskId = &enum.LHDeskId{}
	POS_PRICE = make(map[int32]float32)
	POS_PRICE[0] = 2.00
	POS_PRICE[1] = 8.00
	POS_PRICE[2] = 2.00
	DeskId.DeskId = 100
	DeskId.SSLock = new(sync.RWMutex)
	DeskInit = make(map[int32]map[int32][]int32)

	for kRoomid, _ := range LongHuGameRoom.Rooms {
		DeskInit[kRoomid] = make(map[int32][]int32)
		for i := 1; i < enum.INIT_DESK_NUM+1; i++ {
			var allResult []int32
			for j := 0; j < enum.ALLRESULTNUM; j++ {
				allResult = append(allResult, rand.Int31n(2))
			}
			DeskInit[kRoomid][int32(i)] = allResult
		}
	}

}
func GameRoomInit() *enum.LHGameRoom {
	return &enum.LHGameRoom{
		Rooms:  make(map[int32]*enum.RoomData),
		SSLock: new(sync.RWMutex),
	}
}

func add_desk_id() {
	DeskId.SSLock.Lock()
	DeskId.DeskId = DeskId.DeskId + 1
	DeskId.SSLock.Unlock()
}

func CreatDesk(RoomId int32, deskId int32) (int32, string) {
	PerRoundSId := "001" + string(RoomId) + string(deskId) + string(time.Now().Unix()) //

	var AllResult []int32
	AllResult = DeskInit[RoomId][deskId]
	if _, ok := LongHuGameRoom.Rooms[RoomId]; ok {
		deskInfo := &enum.DeskInfo{
			DeskId:      deskId,
			RoomId:      RoomId,
			PerRoundId:  PerRoundSId,
			Status:      0,
			SuperResult: -1,
			AllResult:   AllResult,
			NextTime:    time.Now().Unix() + enum.READY_TO_BET_SECOND,
			PlayerList:  make(map[string]*enum.Player),
			PlayerBet:   make(map[string]map[int32]int64),
			BeforeBet:   make(map[string]*enum.BeforeBetInfo),
			DeskBet:     make(map[int32]int64),
			SSLock:      new(sync.RWMutex),
		}
		LongHuGameRoom.SSLock.Lock()
		LongHuGameRoom.Rooms[RoomId].Desks[deskId] = deskInfo
		do_new_desk_bet(RoomId, deskId)
		LongHuGameRoom.SSLock.Unlock()

		ticker := time.NewTicker(time.Second * 1)
		count := 0
		go func() {
			for {
				<-ticker.C
				count++
				if count == enum.READY_TO_BET_SECOND {
					go betTimer(deskId, RoomId)
					ticker.Stop()
					runtime.Goexit()
				}
			}
		}()

		fmt.Println("创建的桌子信息----", &LongHuGameRoom.Rooms)
		return deskId, PerRoundSId
	} else {
		return 0, "false"
	}
}

//func CreatNewDesk(RoomId int32, PlayerList map[string]*enum.Player, PlayerBet map[string]map[int32]int64, BeforeBet map[string]*enum.BeforeBetInfo., DeskBet map[int32]int64, ResultList []int32, ThisDeskResult int32) (int32, string) {
//
//	PerRoundSId := "01" + "001" + string(PerRoundId.PerRoundId) //
//
//	if _, ok := LongHuGameRoom.Rooms[RoomId]; ok {
//		deskInfo := &enum.DeskInfo{
//			DeskId:        DeskId.DeskId,
//			RoomId:        RoomId,
//			PerRoundId:    PerRoundSId,
//			PerRoundIdInt: PerRoundId.PerRoundId,
//			Status:        2,
//			SuperResult:   ThisDeskResult,
//			AllResult:     ResultList,
//			NextTime:      time.Now().Unix() + enum.RESULT_TO_READY_SECOND,
//			PlayerList:    PlayerList,
//			PlayerBet:     PlayerBet,
//			BeforeBet:     BeforeBet,
//			DeskBet:       DeskBet,
//			SSLock:        new(sync.RWMutex),
//		}
//		LongHuGameRoom.Rooms[RoomId].SSLock.Lock()
//		LongHuGameRoom.Rooms[RoomId].Desks[DeskId.DeskId] = deskInfo
//		do_new_desk_bet(RoomId,DeskId.DeskId)
//		LongHuGameRoom.Rooms[RoomId].SSLock.Unlock()
//		go resultTimer(DeskId.DeskId, RoomId)
//
//		fmt.Println("创建的桌子信息----", &LongHuGameRoom.Rooms)
//		defer add_desk_id_and_per_round_id()
//		return DeskId.DeskId, PerRoundSId
//	} else {
//		return 0, "false"
//	}
//
//}

func betTimer(deskId int32, RoomId int32) {
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Lock()
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].Status = 1
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].NextTime = time.Now().Unix() + enum.BET_TO_RESULT_SECOND
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Unlock()
	//0->1 把改变状态的通知下发给 在房间里的玩家
	change_desk_state(RoomId, deskId, 1, enum.BET_TO_RESULT_SECOND)
	////////////////////////////////////////////
	ticker := time.NewTicker(time.Second * 1)
	count := 0
	go func() {
		for {
			<-ticker.C
			count++
			if count == enum.BET_TO_RESULT_SECOND {
				go resultTimer(deskId, RoomId)
				ticker.Stop()
				runtime.Goexit()
			}
		}
	}()
}

func resultTimer(deskId int32, RoomId int32) {
	//未完成 判断当前桌子是否有玩家存在，是否有玩家下过住 如果都不存在就销毁
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Lock()
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].Status = 2
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].NextTime = time.Now().Unix() + enum.RESULT_TO_READY_SECOND
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Unlock()
	//1->2 把改变状态的通知下发给 在房间里的玩家
	change_desk_state(RoomId, deskId, 2, enum.RESULT_TO_READY_SECOND)
	////////////////////////////
	var sidlist []string
	for k1, _ := range LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerBet {
		sidlist = append(sidlist, k1)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client := GoClient()
	request := cproto.StorageReq{Uids: sidlist, GameId: 1}
	res, err := client.GetStorageInfo(ctx, &request)
	if err != nil {
		log.ZapLog.With(zap.Error(err)).Error("grpc dial result")
	}
	//未完成 从中心服务器取数据 生成以下3个数据
	SidState := res.UserCtrls
	GamesInfo := res.StoreInfo
	GamesConfig := res.StoreCfg

	StockState, newresult, long, hu := make_result(deskId, RoomId, GamesInfo, GamesConfig)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Lock()
	Settlement := settlement(SidState, deskId, RoomId, &StockState, newresult, GamesInfo, GamesConfig)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].AllResult = append(LongHuGameRoom.Rooms[RoomId].Desks[deskId].AllResult, newresult)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Unlock()
	//未完成 计算开奖结果 给每个人返奖
	a := &bytes.Buffer{}
	resEncoder := json.NewEncoder(a)
	var resjson enum.Resjson
	resjson.Re = newresult
	resjson.Long = long
	resjson.Hu = hu
	resEncoder.Encode(resjson)

	c := &bytes.Buffer{}
	resEncoderC := json.NewEncoder(c)
	var resjsonC []*enum.UserResJson

	var Bets []*cproto.BetSummaryUserBet
	var Winlose []*cproto.WinloseSummaryUserWinlose
	var tableName string
	var tmpGameId uint32 = 1
	var totalWin int64 = 0
	tmpRoomId := uint32(RoomId)
	for sid, v := range Settlement {
		request1 := cproto.ChangeBalanceReq{Uid: sid, Gold: v.AddGold, ChangeType: 2, PerRoundSid: &LongHuGameRoom.Rooms[RoomId].Desks[deskId].PerRoundId, GameId: &tmpGameId, RoomId: &tmpRoomId}
		res1, err1 := client.AddBalance(ctx, &request1)
		if err1 != nil {
			log.ZapLog.With(zap.Error(err1)).Error("grpc dial result")
		}
		var tmp1 *cproto.BetSummaryUserBet
		var tmp2 *cproto.WinloseSummaryUserWinlose
		tmp1.SId = sid
		tmp1.Bet = v.AllBets
		tmp2.SId = sid
		tmp2.Gold = v.AddGold
		tmp2.Tax = v.PlayerServiceCharge
		Bets = append(Bets, tmp1)
		Winlose = append(Winlose, tmp2)

		var tmp3 *enum.UserResJson
		tmp3.P_state = v.StockState
		tmp3.Win = v.Win
		tmp3.Sid = sid
		tmp3.Re = newresult
		tableName = "log_user_per_round_" + sid[len(sid)-1:]
		var memberjson *enum.Memberjson
		for k1, v1 := range LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerBet[sid] {
			memberjson.Pos = k1
			memberjson.Value = v1
		}
		tmp3.Bets = append(tmp3.Bets, memberjson)
		b := &bytes.Buffer{}
		encoder := json.NewEncoder(b)
		encoder.Encode(tmp3)

		resjsonC = append(resjsonC, tmp3)

		onedata := new(db_log.LogUserPerRound)
		onedata.UserSid = sid
		onedata.PerRoundSid = LongHuGameRoom.Rooms[RoomId].Desks[deskId].PerRoundId
		onedata.GameId = 1
		onedata.RoomId = int(RoomId)
		onedata.Change = v.AddGold
		onedata.EndTime = int(time.Now().Unix())
		onedata.Bets = b.String()
		onedata.Result = a.String()
		onedata.PerRoundState = int(v.StockState)
		onedata.Win = v.Win
		onedata.BeforeMoney = LongHuGameRoom.Rooms[RoomId].Desks[deskId].BeforeBet[sid].BeforeBet
		onedata.AfterMoney = *res1.AfterGold
		onedata.Platform = *LongHuGameRoom.Rooms[RoomId].Desks[deskId].BeforeBet[sid].Platform
		onedata.Agent = *LongHuGameRoom.Rooms[RoomId].Desks[deskId].BeforeBet[sid].Agent
		onedata.PlayerServiceCharge = v.PlayerServiceCharge
		_, err := mysql.S1().Table(tableName).Insert(onedata)
		if err != nil {
			log.ZapLog.With(zap.Error(err)).Error("grpc dial result")
		}
		totalWin += v.Win
		//把消息发给客户端
		send_result_msg_to_client(sid, RoomId, deskId, newresult, long, hu, 1, v.AddGold, *res1.AfterGold)
	}
	for tmpsid, _ := range LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerList {
		if _, ok := Settlement[tmpsid]; ok == false {
			send_result_msg_to_client(tmpsid, RoomId, deskId, newresult, long, hu, 0, 0, 0)
		}
	}

	resEncoderC.Encode(resjsonC)
	if len(sidlist) != 0 {
		logLongHu := new(db_log.Log1PerRoundLonghudou)
		logLongHu.PerRoundSid = LongHuGameRoom.Rooms[RoomId].Desks[deskId].PerRoundId
		logLongHu.RoomId = int(RoomId)
		logLongHu.DataTime = int(time.Now().Unix())
		logLongHu.UsersData = c.String()
		logLongHu.Result = a.String()
		if totalWin >= 0 {
			logLongHu.ResultState = -1
		} else {
			logLongHu.ResultState = 1
		}
		logLongHu.Amount = totalWin
		var longhu_table_name string = "log_1_per_round_longhudou"
		mysql.S1().Table(longhu_table_name).Insert(logLongHu)
	}

	request2 := cproto.BetSummary{Bets: Bets, GameId: 1}
	_, err2 := client.UserBetSummary(ctx, &request2)
	if err2 != nil {
		log.ZapLog.With(zap.Error(err2)).Error("grpc dial result")
	}
	request3 := cproto.WinloseSummary{Winlose: Winlose, GameId: 1}
	_, err3 := client.UserWinloseSummary(ctx, &request3)

	if err3 != nil {
		log.ZapLog.With(zap.Error(err3)).Error("grpc dial result")
	}
	//计时器处理
	ticker := time.NewTicker(time.Second * 1)
	count := 0
	go func() {
		for {
			<-ticker.C
			count++
			if count == enum.RESULT_TO_READY_SECOND {
				go startWaitTimer(deskId, RoomId)
				ticker.Stop()
				runtime.Goexit()
			}
		}
	}()
}
func startWaitTimer(deskId int32, RoomId int32) {
	PerRoundSId := "001" + string(RoomId) + string(deskId) + string(time.Now().Unix()) //

	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Lock()
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].DeskBet = make(map[int32]int64)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerBet = make(map[string]map[int32]int64)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].BeforeBet = make(map[string]*enum.BeforeBetInfo)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].Status = 0
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SuperResult = -1
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].NextTime = time.Now().Unix() + enum.READY_TO_BET_SECOND
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].PerRoundId = PerRoundSId
	do_new_desk_bet(RoomId, deskId)
	LongHuGameRoom.Rooms[RoomId].Desks[deskId].SSLock.Unlock()
	//2->0 把改变状态的通知下发给 在房间里的玩家 通知倒计时
	new_per_round(RoomId, deskId, enum.READY_TO_BET_SECOND)
	///////////////////////////////////////////////

	ticker := time.NewTicker(time.Second * 1)
	count := 0
	go func() {
		for {
			<-ticker.C
			count++
			if count == enum.READY_TO_BET_SECOND {
				go betTimer(deskId, RoomId)
				ticker.Stop()
				runtime.Goexit()
			}
		}
	}()
}

func AddNewResult() {
	ticker := time.NewTicker(time.Second * 20)
	go func() {
		for {
			<-ticker.C
			for roomid, desks := range DeskInit {
				for deskid, allResult := range desks {
					allResult = allResult[1:]
					allResult = append(allResult, rand.Int31n(2))
					DeskInit[roomid][deskid] = allResult
				}
			}
		}
	}()
}

func change_desk_state(RoomId int32, deskId int32, status int32, nextTime int) {
	var uids []string
	for sid, _ := range LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerList {
		fmt.Println("change_desk_state--0--", sid, deskId)
		uids = append(uids, sid)
	}
	fmt.Println("change_desk_state--1--", deskId)
	newnextTime := int32(nextTime)
	Info := protoStruct.MGame_1ChangeStateToc{
		Status:   &status,
		NextTime: &newnextTime,
	}
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: "",
		BAllUser: false,
		Uids:     uids,
		Msg:      uint32(enum.CMD_GAME_1_CHANGE_STATE),
		Data:     data,
	}
	Stream.GrpcSendClientData <- &sendCMsg
}
func new_per_round(RoomId int32, deskId int32, nextTime int) {
	var uids []string
	for sid, _ := range LongHuGameRoom.Rooms[RoomId].Desks[deskId].PlayerList {
		uids = append(uids, sid)
	}
	newnextTime := int32(nextTime)
	var state int32 = 0
	var area []*protoStruct.PGame_1BetsArea
	number := LongHuGameRoom.Rooms[RoomId].Desks[deskId].PerRoundId
	var myBets int64 = 0
	for k, v := range POS_PRICE {
		var tmp = &protoStruct.PGame_1BetsArea{}
		tmp2 := LongHuGameRoom.Rooms[RoomId].Desks[deskId].DeskBet[k]
		tmp.Area = &k
		tmp.Odds = &v
		tmp.MyBets = &myBets
		tmp.AllBets = &tmp2
		area = append(area, tmp)
	}
	Info := protoStruct.MGame_1StartNewBetsToc{
		Number:   &number,
		Status:   &state,
		NextTime: &newnextTime,
		Area:     area,
	}
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: "",
		BAllUser: false,
		Uids:     uids,
		Msg:      uint32(enum.CMD_GAME_1_START_NEW_BETS),
		Data:     data,
	}
	Stream.GrpcSendClientData <- &sendCMsg
}

func send_result_msg_to_client(sid string, RoomId int32, deskId int32, newresult int32, long int32, hu int32, isbet int32, addGold int64, afterGold int64) {
	Info := protoStruct.MGame_1EndResultToc{
		Room:       &RoomId,
		Desk:       &deskId,
		Result:     &newresult,
		Long:       &long,
		Hu:         &hu,
		Isbat:      &isbet,
		MyAward:    &addGold,
		AfterMoney: &afterGold,
	}
	var uids []string
	uids = append(uids, sid)
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: "",
		BAllUser: false,
		Uids:     uids,
		Msg:      uint32(enum.CMD_GAME_1_END_RESULT),
		Data:     data,
	}
	Stream.GrpcSendClientData <- &sendCMsg
}
