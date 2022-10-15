package enum

const (
	INIT_DESK_NUM          = 4  // 初始化虚拟桌子
	ALLRESULTNUM           = 30 // 录单个数
	READY_TO_BET_SECOND    = 5  // 准备到下注的时间间隔
	BET_TO_RESULT_SECOND   = 10 // 下注到开奖的时间间隔
	RESULT_TO_READY_SECOND = 5  // 开奖到准备的时间间隔
)
const (
	PM = 1 // 平民场
	XZ = 2 // 小资场
	LB = 3 // 老板场
	TH = 4 // 土豪场
)
const (
	CMD_ERROR                 = 99
	CMD_GAME_1_ENTER_GAME     = 1009
	CMD_GAME_1_ENTER_GAME_REQ = 1001
	CMD_GAME_1_ENTER_ROOM     = 1002
	CMD_GAME_1_ENTER_DESK     = 1003
	CMD_GAME_1_LEAVE_DESK     = 1004
	CMD_GAME_1_BETS           = 1005
	CMD_GAME_1_CHANGE_STATE   = 1006
	CMD_GAME_1_END_RESULT     = 1007
	CMD_GAME_1_START_NEW_BETS = 1008

	CMDS string = "1001,1002,1003,1004,1005,1006,1007,1008,1009"
)
