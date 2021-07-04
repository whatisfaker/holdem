package holdem

import "time"

type Reciever interface {
	//ID 用户识别
	ID() string
	//ErrorOccur 接收错误
	ErrorOccur(int, error)
	//RoomerGameInformation 游戏信息
	RoomerGameInformation(*HoldemState)
	//RoomerJoin 接收有人进入游戏
	RoomerJoin(string)
	//RoomerLeave 接收有人离开游戏
	RoomerLeave(string)
	//RoomerSeated 接收有人坐下
	RoomerSeated(int8, string, PlayType)
	//RoomerRoomerStandUp
	RoomerStandUp(int8, string, int8)
	//RoomerGetCard 接收有人收到牌（位置,牌数量,操作者)
	RoomerGetCard([]int8, int8, *StartNewHandInfo, *Operator)
	//RoomerGetPublicCard 接收公共牌(牌,谁操作,是否是你操作)
	RoomerGetPublicCard([]*Card, *Operator, bool)
	//RoomerGetAction 接收有人动作（按钮位, 位置，动作，金额(如果下注), 是否是你)
	RoomerGetAction(int8, int8, ActionDef, uint, *Operator, bool)
	//RoomerGetBuyInsurance 接收谁购买了保险的信息
	RoomerGetBuyInsurance(seat int8, buy []*BuyInsurance, round Round)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards([]*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult([]*Result)
	//RoomerKeepSeat 接收有人占座(座位号)
	RoomerKeepSeat(int8, time.Duration)
	//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注),下一个操作者)
	PlayerActionSuccess(int8, int8, ActionDef, uint, *Operator)
	//PlayerGetCard 玩家获得自己发到的牌（座位号,牌,发牌顺序,几张牌,下一个操作者是否是你)
	PlayerGetCard(int8, []*Card, []int8, int8, *StartNewHandInfo, *Operator, bool)
	//PlayerCanNotBuyInsurance 玩家无法购买保险(座位号,outs数量,回合)
	PlayerCanNotBuyInsurance(seat int8, outsLen int, round Round)
	//PlayerCanBuyInsurance 玩家可以开始买保险(座位号,outs数量,赔率,具体outs,回合)
	PlayerCanBuyInsurance(seat int8, outsLen int, odds float64, outs map[int8][]*UserOut, round Round)
	//PlayerBuyInsuranceSuccess 玩家购买保险成功（座位号，金额）
	PlayerBuyInsuranceSuccess(seat int8, buy []*BuyInsurance)
	//PlayerBringInSuccess 玩家带入成功
	PlayerBringInSuccess(seat int8, chip uint)
	//PlayerJoinSuccess 玩家进入游戏成功
	PlayerJoinSuccess(string, *HoldemState)
	//PlayerLeaveSuccess 玩家离开游戏成功
	PlayerLeaveSuccess(string)
	//PlayerSeatedSuccess 玩家坐下成功(补盲状态)
	PlayerSeatedSuccess(int8, PlayType)
	//PlayerCanPayToPlay 玩家可以补盲了
	PlayerCanPayToPlay(int8)
	//PlayerPayToPlaySuccesss 玩家补盲成功
	PlayerPayToPlaySuccesss(int8)
	//PlayerReadyStandUpSuccess 玩家准备站起成功
	PlayerReadyStandUpSuccess(int8)
	//PlayerStandUp 玩家站起
	PlayerStandUp(int8, int8)
	//PlayerKeepSeat 玩家占座(座位号)
	PlayerKeepSeat(int8, time.Duration)
}
