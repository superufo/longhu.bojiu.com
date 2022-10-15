package enum

import (
	db_server "common.bojiu.com/models/bj_server"
	protoStruct "longhu.bojiu.com/internal/gstream/proto"
	"sync"
)

type Player struct {
	UserInfo *protoStruct.MUserInfo
	User     *protoStruct.MUser
}

type LHDeskId struct {
	DeskId int32 // int 场次
	SSLock *sync.RWMutex
}

type LHPerRoundId struct {
	PerRoundId int32 // int 场次
	SSLock     *sync.RWMutex
}

type LHGameRoom struct {
	Rooms  map[int32]*RoomData // int 场次
	SSLock *sync.RWMutex
}

// SessionData 场次信息
type RoomData struct {
	State  int32
	MaxBet int64
	MinBet int64
	Desks  map[int32]*DeskInfo
	SSLock *sync.RWMutex
}

type DeskInfo struct {
	DeskId      int32                      // 桌号
	RoomId      int32                      // 房间号
	PerRoundId  string                     // 牌局号
	Status      int32                      // 当前牌桌的状态   0准备阶段  1下注中  2开牌
	SuperResult int32                      // 当前牌桌特殊开奖结果 正常  -1  龙 0，和1，虎2
	AllResult   []int32                    //录单v
	NextTime    int64                      // 下次状态变化的时间
	PlayerList  map[string]*Player         // 玩家信息
	PlayerBet   map[string]map[int32]int64 // 玩家投注区域的筹码 make(map[玩家Sid]map[0,1,2]int64)
	BeforeBet   map[string]*BeforeBetInfo  // 记录当局开始之前玩家剩余多少钱
	DeskBet     map[int32]int64            // 012 的下注
	SSLock      *sync.RWMutex
}

//需要临时存放的数据
type BeforeBetInfo struct {
	BeforeBet int64   //玩家
	Platform  *string //用户渠道
	Agent     *string //用户代理
}

//发送给中心数据服务的
type SidBetList struct {
	SidBet []SidBet //玩家
}

type SidBet struct {
	Sid string //玩家id
	Bet int64  //玩家总下注
}
type SidState struct {
	Sid   string //玩家id
	State int32  //玩家控制状态
}

//中心数据服务返回的
type Relist struct {
	SidState    []SidState           //玩家id
	GamesInfo   *db_server.GamesInfo //玩家控制状态
	GamesConfig *db_server.GamesConfig
}

type TmpSettlement struct {
	Win                 int64
	AddGold             int64
	AllBets             int64
	PlayerServiceCharge int64
	StockState          int32
}

type Memberjson struct {
	Pos   int32 `json:"pos"`
	Value int64 `json:"value"`
}

type Resjson struct {
	Re   int32 `json:"re"`
	Long int32 `json:"Long"`
	Hu   int32 `json:"Hu"`
}

type UserResJson struct {
	Sid     string        `json:"sid"`
	Re      int32         `json:"re"`
	P_state int32         `json:"p_state"`
	Win     int64         `json:"win"`
	Bets    []*Memberjson `json:"bets"`
}
