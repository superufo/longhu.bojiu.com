package server

import (
	"context"
	"fmt"
	toolProto "github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"longhu.bojiu.com/enum"
	"longhu.bojiu.com/internal/gstream/pb"
	"longhu.bojiu.com/internal/gstream/proto"
	cproto "longhu.bojiu.com/internal/gstream/proto"
	protoStruct "longhu.bojiu.com/internal/proto"
	"longhu.bojiu.com/pkg/log"
	"time"
)

func GoClient() proto.StorageClient {
	//log.ZapLog = log.InitLogger()
	//scfg := config.NewServerCfg()
	//log.ZapLog.Info(fmt.Sprintf("%s:%d", scfg.GetIp(), scfg.GetPort()))

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", "127.0.0.1", 19001), grpc.WithInsecure())
	if err != nil {
		log.ZapLog.With(zap.Error(err)).Error("grpc dial error")
	}

	return proto.NewStorageClient(conn)
}

func doEnterDesk(roomId int32, deskId int32, sid string, ClientId string) pb.StreamResponseData {
	//未完成 所有client 的销毁
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client := GoClient()
	request := cproto.UserRequest{Uid: sid}
	res, _ := client.GetUserInfo(ctx, &request)
	request1 := cproto.UserGameInfo{SId: sid, GameId: 1, RoomId: int64(roomId), DeskId: int64(deskId)}
	client.SetUserGameInfo(ctx, &request1)
	var player enum.Player
	player.UserInfo = res.UserInfo
	player.User = res.User
	LongHuGameRoom.Rooms[roomId].Desks[deskId].SSLock.Lock()
	LongHuGameRoom.Rooms[roomId].Desks[deskId].PlayerList[sid] = &player
	LongHuGameRoom.Rooms[roomId].Desks[deskId].SSLock.Unlock()
	var betsArea []*protoStruct.PGame_1BetsArea
	for k, v := range LongHuGameRoom.Rooms[roomId].Desks[deskId].DeskBet {
		tmp0 := LongHuGameRoom.Rooms[roomId].Desks[deskId].PlayerBet[sid][k]
		tmp1 := POS_PRICE[k]
		var tmp protoStruct.PGame_1BetsArea
		tmp.Area = &k
		tmp.AllBets = &v
		tmp.Odds = &tmp1
		tmp.MyBets = &tmp0
		betsArea = append(betsArea, &tmp)
	}
	n := LongHuGameRoom.Rooms[roomId].Desks[deskId].NextTime - time.Now().Unix()
	nt := int32(n)
	Info := protoStruct.MGame_1EnterDeskToc{
		DeskId:    &deskId,
		Number:    &LongHuGameRoom.Rooms[roomId].Desks[deskId].PerRoundId,
		Status:    &LongHuGameRoom.Rooms[roomId].Desks[deskId].Status,
		NextTime:  &nt,
		AllResult: LongHuGameRoom.Rooms[roomId].Desks[deskId].AllResult,
		BetsArea:  betsArea,
	}
	data, _ := toolProto.Marshal(&Info)

	sendCMsg := pb.StreamResponseData{
		ClientId: ClientId,
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_ENTER_DESK),
		Data:     data,
	}
	return sendCMsg
}
func makeErrorData(sid string, err int32, ClientId string) pb.StreamResponseData {
	Info := protoStruct.MErrorToc{
		ErrorCode: &err,
	}
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: ClientId,
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_ERROR),
		Data:     data,
	}
	return sendCMsg
}

func doLeaveDesk(ClientId string, roomId int32, deskId int32) pb.StreamResponseData {
	Info := protoStruct.MGame_1LeaveDeskToc{
		RoomId: &roomId,
		DeskId: &deskId,
	}
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: ClientId,
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_LEAVE_DESK),
		Data:     data,
	}
	return sendCMsg
}

func doPlayerBet(sid string, area int32, chip int64, roomid int32, deskId int32, ClientId string) pb.StreamResponseData {
	if chip <= 0 {
		return makeErrorData(sid, 1, ClientId)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client := GoClient()
	var tmpGameId uint32 = 1
	tmpRoomId := uint32(roomid)
	request := cproto.ChangeBalanceReq{Uid: sid, Gold: chip, ChangeType: 1, PerRoundSid: &LongHuGameRoom.Rooms[roomid].Desks[deskId].PerRoundId, GameId: &tmpGameId, RoomId: &tmpRoomId}
	res, _ := client.ReduceBalance(ctx, &request)
	if res.Code == 0 {
		if _, ok := LongHuGameRoom.Rooms[roomid].Desks[deskId].BeforeBet[sid]; ok {
			return doPlayerBet_1(sid, area, chip, roomid, deskId, *res.AfterGold, ClientId)
		} else {
			doPlayerBetBefore(sid, *res.BeforeGold, roomid, deskId)
			return doPlayerBet_1(sid, area, chip, roomid, deskId, *res.AfterGold, ClientId)
		}
	} else {
		return makeErrorData(sid, 1, ClientId)
	}
}
func doPlayerBetBefore(sid string, beforeGold int64, roomid int32, deskId int32) {
	LongHuGameRoom.Rooms[roomid].Desks[deskId].SSLock.Lock()
	var tmp enum.BeforeBetInfo
	tmp.BeforeBet = beforeGold
	tmp.Platform = LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerList[sid].User.Platform
	tmp.Agent = LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerList[sid].User.Agent
	LongHuGameRoom.Rooms[roomid].Desks[deskId].BeforeBet[sid] = &tmp
	LongHuGameRoom.Rooms[roomid].Desks[deskId].SSLock.Unlock()
}
func doPlayerBet_1(sid string, area int32, chip int64, roomid int32, deskId int32, afterGold int64, ClientId string) pb.StreamResponseData {
	var myAllChip int64
	LongHuGameRoom.Rooms[roomid].Desks[deskId].SSLock.Lock()
	if _, ok := LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid]; ok {
		if val, ok := LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid][area]; ok {
			LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid][area] = val + chip
			myAllChip = val + chip
		} else {
			myAllChip = chip
			LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid][area] = chip
		}
	} else {
		myAllChip = chip
		LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid] = make(map[int32]int64)
		LongHuGameRoom.Rooms[roomid].Desks[deskId].PlayerBet[sid][area] = chip
	}
	LongHuGameRoom.Rooms[roomid].Desks[deskId].SSLock.Unlock()

	Info := protoStruct.MGame_1BetsToc{
		Area:      &area,
		MyAllChip: &myAllChip,
		MyChip:    &afterGold,
		Chip:      &chip,
	}
	data, _ := toolProto.Marshal(&Info)
	sendCMsg := pb.StreamResponseData{
		ClientId: ClientId,
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_BETS),
		Data:     data,
	}
	return sendCMsg
}
