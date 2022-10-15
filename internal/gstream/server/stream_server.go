package server

import (
	Utils "common.bojiu.com/utils"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"io"
	"longhu.bojiu.com/config"
	"longhu.bojiu.com/enum"
	"longhu.bojiu.com/internal/gstream/pb"
	protoStruct "longhu.bojiu.com/internal/proto"
	"longhu.bojiu.com/pkg/log"
	"net"
	"strings"
)

// var center  = storage.StorageServerImpl
var Stream *streamServer

type streamServer struct {
	GrpcRecvClientData chan *pb.StreamRequestData
	GrpcSendClientData chan *pb.StreamResponseData
}

func NewStreamServer() *streamServer {
	Stream = &streamServer{
		make(chan *pb.StreamRequestData, 100),
		make(chan *pb.StreamResponseData, 100),
	}
	//
	return Stream
}

//func init() {
//	GrpcRecvClientData = make(chan *pb.StreamRequestData, 100)
//	GrpcSendClientData = make(chan *pb.StreamResponseData, 100)
//}

// PPStream log.ZapLog.With(zap.Any("err", err)).Error("收到网关数据错误")
func (gs *streamServer) PPStream(stream pb.ForwardMsg_PPStreamServer) error {
	stop := make(chan struct{})
	defer func() {
		if e := recover(); e != nil {
			log.ZapLog.Info("PPStream recover", zap.Any("err", e.(error)))
		}
		close(stop)
	}()
	go gs.response(stream, stop)
	go gs.dispatch(stop)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.ZapLog.Info("PPStream recv io EOF", zap.Any("err", err))
			return nil
		}
		if err != nil {
			log.ZapLog.Info("PPStream recv error", zap.Any("err", err))
			return err
		}
		info := fmt.Sprintf("收到网关数据:协议号=%+v,加密字符=%+v,随机字符=%+v,protobuf=%+v", msg.GetMsg(), Utils.ToHexString(msg.GetSecret()), msg.GetSerialNum(), msg.GetData())
		log.ZapLog.Info(info)
		gs.GrpcRecvClientData <- msg
	}
}

func (gs *streamServer) response(stream pb.ForwardMsg_PPStreamServer, stop chan struct{}) {
	defer func() {
		if e := recover(); e != nil {
			log.ZapLog.Info("stream response", zap.Any("err", e.(error)))
		}
	}()
	for {
		select {
		case sd := <-gs.GrpcSendClientData:
			//业务代码
			if err := stream.Send(sd); err != nil {
				log.ZapLog.Info("", zap.Any("发给网关失败err", err))
			} else {
				log.ZapLog.Info("", zap.Any("发给网关成功msg", sd.String()))
			}
		case <-stop:
			return
		}
	}
}

func (gs *streamServer) dispatch(stop chan struct{}) {
	defer func() {
		if e := recover(); e != nil {
			log.ZapLog.Info("stream dispatch", zap.Any("err", e.(error)))
		}
	}()
	for {
		select {
		case cmsg := <-gs.GrpcRecvClientData:
			{
				log.ZapLog.Info("dispatch", zap.Any("Msg", cmsg.Msg))
				if !strings.Contains(enum.CMDS, fmt.Sprintf("%d", cmsg.Msg)) {
					log.ZapLog.Error("不存在的消息", zap.Any("msg", cmsg.Msg))
				}
				if err := gs.handlerMsg(cmsg); err != nil {
					log.ZapLog.Info("handlerMsg error ", zap.Any("Msg", cmsg.Msg), zap.Any("err", err))
				}
			}
		case <-stop:
			return
		}
	}
}

// 消息处理
func (gs *streamServer) handlerMsg(clientMsg *pb.StreamRequestData) error {
	// 判断游戏是否存在
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_ENTER_GAME {
		gs.enterGame(clientMsg)
	}

	// 进入游戏，回复房间信息
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_ENTER_GAME_REQ {
		gs.enterGameReq(clientMsg)
	}

	// 选择房间，回复桌子信息
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_ENTER_ROOM {
		gs.enterRoom(clientMsg)
	}

	// 选择桌子，回复桌子信息
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_ENTER_DESK {
		gs.enterDesk(clientMsg)
	}

	// 玩家下注，返回信息
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_BETS {
		gs.playerBet(clientMsg)
	}

	// 玩家退出游戏，离开桌子
	if uint16(clientMsg.Msg) == enum.CMD_GAME_1_LEAVE_DESK {
		gs.playerOut(clientMsg)
	}

	return nil
}

//
//// sendToClientRes 告知游戏结果
//func sendToClientRes(sessionId, deskId int32, playerList map[string]*enum.UserInfo) {
//
//}
//
//// sendToClientChangeStatus  告知客户端状态改变
//func sendToClientChangeStatus(sessionId, nextStatus, time, deskId int32, playerList map[string]*enum.UserInfo) {
//
//}
// ENTER_GAME 进入游戏
func (gs *streamServer) enterGame(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1EnterGameTos{}

	if err := proto.Unmarshal(msg.Data, &l); err != nil {
		log.ZapLog.With(zap.Any("err", err)).Error("enterDesk proto3解码错误")
		return errors.New("proto3解码错误")
	}
	var info protoStruct.MGame_1EnterGameToc
	info.GameId = l.GameId
	info.Room = l.Room
	info.Desk = l.Desk
	data, _ := proto.Marshal(&info)
	sendCMsg := pb.StreamResponseData{
		ClientId: msg.GetClientId(),
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_ENTER_GAME),
		Data:     data,
	}
	gs.GrpcSendClientData <- &sendCMsg
	fmt.Println("-------------------------------------", l.GameId)
	return nil
}

// ENTER_GAME 进入游戏
func (gs *streamServer) enterGameReq(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1EnterGameReqTos{}

	if err := proto.Unmarshal(msg.Data, &l); err != nil {
		log.ZapLog.With(zap.Any("err", err)).Error("enterDesk proto3解码错误")
		return errors.New("proto3解码错误")
	}
	var rooms protoStruct.MGame_1EnterGameReqToc
	var room protoStruct.PRoom_1RoomInfo
	for kRoomId, v := range LongHuGameRoom.Rooms {
		room = protoStruct.PRoom_1RoomInfo{
			RoomId: &kRoomId,
			State:  &v.State,
		}
		rooms.Room = append(rooms.Room, &room)
	}
	data, _ := proto.Marshal(&rooms)
	sendCMsg := pb.StreamResponseData{
		ClientId: msg.GetClientId(),
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_ENTER_GAME_REQ),
		Data:     data,
	}
	gs.GrpcSendClientData <- &sendCMsg
	fmt.Println("-------------------------------------", l.GameId)
	return nil
}

// CHOOSE_SESSION 选择场次
func (gs *streamServer) enterRoom(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1EnterRoomTos{}
	if err := proto.Unmarshal(msg.Data, &l); err != nil {
		log.ZapLog.With(zap.Any("err", err)).Error("enterDesk proto3解码错误")
		return errors.New("proto3解码错误")
	}
	var deskNum int32 = 0
	var desks protoStruct.MGame_1EnterRoomToc
	var desk protoStruct.PRoom_1DeskInfo
	for _, deskInfo := range LongHuGameRoom.Rooms[l.GetRoomId()].Desks {
		deskNum += 1
		desk = protoStruct.PRoom_1DeskInfo{
			DeskId:    &deskInfo.DeskId,
			State:     &deskInfo.Status,
			AllResult: deskInfo.AllResult,
		}
		desks.Desk = append(desks.Desk, &desk)
		if deskNum >= enum.INIT_DESK_NUM {
			break
		}
	}
	if deskNum < enum.INIT_DESK_NUM {
		for kdeskid, vdeskresult := range DeskInit[l.GetRoomId()] {
			var state int32 = 1
			desk = protoStruct.PRoom_1DeskInfo{
				DeskId:    &kdeskid,
				State:     &state,
				AllResult: vdeskresult,
			}
			desks.Desk = append(desks.Desk, &desk)
			deskNum *= 1
			if deskNum >= enum.INIT_DESK_NUM {
				break
			}
		}
	}
	data, _ := proto.Marshal(&desks)
	sendCMsg := pb.StreamResponseData{
		ClientId: msg.GetClientId(),
		BAllUser: false,
		Uids:     nil,
		Msg:      uint32(enum.CMD_GAME_1_ENTER_ROOM),
		Data:     data,
	}
	gs.GrpcSendClientData <- &sendCMsg
	return nil
}

// CHOOSE_DESK  选择桌子
func (gs *streamServer) enterDesk(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1EnterDeskTos{}
	theme := msg.ClientId
	fmt.Println("---------------3----------------------", theme)
	comma := strings.Index(theme, "_")
	sid := theme[:comma]

	if err := proto.Unmarshal(msg.Data, &l); err != nil {
		log.ZapLog.With(zap.Any("err", err)).Error("enterDesk proto3解码错误")
		return errors.New("proto3解码错误")
	}
	fmt.Println("-------------------1------------------", *l.DeskId, *l.RoomId)
	if _, ok := LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId]; ok {
		fmt.Println("----------------2---------------------", *l.DeskId, *l.RoomId)
		sendCMsg := doEnterDesk(*l.RoomId, *l.DeskId, sid, msg.ClientId)
		gs.GrpcSendClientData <- &sendCMsg
	} else {
		if _, ok := LongHuGameRoom.Rooms[*l.RoomId]; ok {
			if _, ok := DeskInit[*l.RoomId][*l.DeskId]; ok {
				//生成指定录单的桌子 然后坐下去
				CreatDesk(*l.RoomId, *l.DeskId)
				fmt.Println("---------------3----------------------", sid)
				sendCMsg := doEnterDesk(*l.RoomId, *l.DeskId, sid, msg.ClientId)
				fmt.Println("-------00000000---------------", *l.DeskId, *l.RoomId)
				gs.GrpcSendClientData <- &sendCMsg
			} else {
				//生成随机的录单的桌子  然后坐下去
				if len(LongHuGameRoom.Rooms[*l.RoomId].Desks) == 0 {
					for tmpDeskid, _ := range DeskInit[*l.RoomId] {
						CreatDesk(*l.RoomId, tmpDeskid)
						fmt.Println("-----------------4--------------------", *l.DeskId, *l.RoomId)
						sendCMsg := doEnterDesk(*l.RoomId, tmpDeskid, sid, msg.ClientId)
						fmt.Println("----------1111111----------------", *l.DeskId, *l.RoomId)
						gs.GrpcSendClientData <- &sendCMsg
						break
					}
				} else {
					//随机找个桌子坐下 没有则生成DeskInit中的桌子
					for tmpDeskid, _ := range LongHuGameRoom.Rooms[*l.RoomId].Desks {
						fmt.Println("-----------------5--------------------", *l.DeskId, *l.RoomId)
						sendCMsg := doEnterDesk(*l.RoomId, tmpDeskid, sid, msg.ClientId)
						fmt.Println("-------222222222----------------", *l.DeskId, *l.RoomId)
						gs.GrpcSendClientData <- &sendCMsg
						break
					}
				}
			}
		} else {
			log.ZapLog.With(zap.Any("err", "room")).Error("room not exit")
		}
	}
	return nil
}

// PLAYER_BET   玩家下注
func (gs *streamServer) playerBet(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1BetsTos{}
	theme := msg.ClientId
	comma := strings.Index(theme, "_")
	sid := theme[:comma]
	if err := proto.Unmarshal(msg.Data, &l); err != nil {
		log.ZapLog.With(zap.Any("err", err)).Error("enterDesk proto3解码错误")
		return errors.New("proto3解码错误")
	}
	fmt.Println("-------------------------------------", l.DeskId, l.Room)
	if _, ok := LongHuGameRoom.Rooms[*l.Room].Desks[*l.DeskId]; ok {
		if LongHuGameRoom.Rooms[*l.Room].Desks[*l.DeskId].Status == 1 {
			sendCMsg := doPlayerBet(sid, *l.Area, *l.Chip, *l.Room, *l.DeskId, msg.ClientId)
			gs.GrpcSendClientData <- &sendCMsg
		} else {
			log.ZapLog.With(zap.Any("err", "state")).Error("state not 1")
			sendCMsg := makeErrorData(sid, 1, msg.ClientId)
			gs.GrpcSendClientData <- &sendCMsg
		}
	} else {
		log.ZapLog.With(zap.Any("err", "desk")).Error("desk not exit")
		sendCMsg := makeErrorData(sid, 1, msg.ClientId)
		gs.GrpcSendClientData <- &sendCMsg
	}
	return nil
}

// 玩家退出游戏
func (gs *streamServer) playerOut(msg *pb.StreamRequestData) error {
	var l = protoStruct.MGame_1LeaveDeskTos{}
	theme := msg.ClientId
	comma := strings.Index(theme, "_")
	sid := theme[:comma]

	if _, ok := LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId]; ok {
		if _, ok := LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId].PlayerList[sid]; ok {
			LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId].SSLock.Lock()
			delete(LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId].PlayerList, sid)
			LongHuGameRoom.Rooms[*l.RoomId].Desks[*l.DeskId].SSLock.Unlock()
		}
	}
	sendCMsg := doLeaveDesk(msg.ClientId, *l.RoomId, *l.DeskId)
	gs.GrpcSendClientData <- &sendCMsg
	return nil
}

func Run() {
	//streamIp := viper.Vp.GetString("ser.stream.ip")
	//streamPort := viper.Vp.GetInt("ser.stream.port")
	var server pb.ForwardMsgServer
	sImpl := NewStreamServer()

	server = sImpl

	g := grpc.NewServer()

	// 2.注册逻辑到server中
	pb.RegisterForwardMsgServer(g, server)

	scfg := config.NewServerCfg()
	instance := fmt.Sprintf("%s:%d", scfg.GetIp(), scfg.GetPort())

	log.ZapLog.With(zap.Any("addr", instance)).Info("Run")
	// 3.启动server
	lis, err := net.Listen("tcp", instance)
	if err != nil {
		panic("监听错误:" + err.Error())
	}

	err = g.Serve(lis)
	if err != nil {
		panic("启动错误:" + err.Error())
	}

	//sImpl.dispatch()
}
