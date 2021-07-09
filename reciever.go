package holdem

import "time"

type Reciever interface {
	//ErrorOccur 接收错误
	ErrorOccur(string, int, error)
	//RoomerMessage 接收消息（uid == "" 代表是全局广播, 否则为来源uid的用户发送的)
	RoomerMessage(hid string, code int, msg interface{}, uid string, seat ...int8)
	//RoomerGameStart 游戏开始
	RoomerGameStart(hid string)
	//RoomerGamePauseResume 游戏暂停/继续
	RoomerGamePauseResume(hid string, pausedOrResume bool)
	//RoomerGameEnd 游戏结束
	RoomerGameEnd(hid string)
	//RoomerPots 当前池
	RoomerGamePots(hid string, pots []*Pot, round Round)
	//RoomerExceedTime 延时
	RoomerExceedTime(hid string, seat int8, uid string, times int8, tm time.Duration)
	//RoomerJoin 接收有人进入游戏
	RoomerJoin(hid string, userID string)
	//RoomerLeave 接收有人离开游戏
	RoomerLeave(hid string, userID string)
	//RoomerSeated 接收有人坐下
	RoomerSeated(hid string, seat int8, userID string, te PlayType)
	//RoomerRoomerStandUp
	RoomerStandUp(hid string, seat int8, userID string, reaonCode int8)
	//RoomerGetCard 接收有人收到牌（位置,牌数量,操作者)
	RoomerGetCard(hid string, reciveSeats []int8, cardsNum int8, handInfo *StartNewHandInfo, op *Operator)
	//RoomerGetPublicCard 接收公共牌(牌,谁操作)
	RoomerGetPublicCard(hid string, cards []*Card, op *Operator)
	//RoomerGetAction 接收有人动作（位置，用户, 动作，金额(如果下注),当前要操作者)
	RoomerGetAction(hid string, seat int8, usreID string, act ActionDef, num uint, op *Operator)
	//RoomerGetWaitInsurance 接收谁开始买保险
	RoomerGetWaitInsurance(hid string, seat int8, uid string, dur time.Duration, round Round)
	//RoomerGetBuyInsurance 接收谁购买了保险的信息
	RoomerGetBuyInsurance(hid string, seat int8, uid string, buy []*BuyInsurance, round Round)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards(hid string, cards []*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult(hid string, res []*Result)
	//RoomerAutoOp 玩家托管(开启/关闭)
	RoomerAutoOp(hid string, seat int8, userID string, open bool)
	//RoomerKeepSeat 接收有人占座(座位号)
	RoomerKeepSeat(hid string, seat int8, userID string, tm time.Duration)
	//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注),下一个操作者)
	PlayerActionSuccess(hid string, seat int8, userID string, act ActionDef, num uint, op *Operator)
	//PlayerGetCard 玩家获得自己发到的牌（座位号,牌,发牌顺序,几张牌,下一个操作者是否是你)
	PlayerGetCard(hid string, seat int8, userID string, cards []*Card, dealOrder []int8, num int8, handsInfo *StartNewHandInfo, op *Operator)
	//PlayerCanNotBuyInsurance 玩家无法购买保险(座位号,outs数量,回合)
	PlayerCanNotBuyInsurance(hid string, seat int8, userID string, outsLen int, round Round)
	//PlayerCanBuyInsurance 玩家可以开始买保险(座位号,outs数量,赔率,具体outs,回合)
	PlayerCanBuyInsurance(hid string, seat int8, userID string, outsLen int, odds float64, outs map[int8][]*UserOut, round Round)
	//PlayerBuyInsuranceSuccess 玩家购买保险成功（座位号，金额）
	PlayerBuyInsuranceSuccess(hid string, seat int8, userID string, buy []*BuyInsurance)
	//PlayerBringInSuccess 玩家带入成功
	PlayerBringInSuccess(hid string, seat int8, userID string, chip uint)
	//PlayerJoinSuccess 玩家进入游戏成功
	PlayerJoinSuccess(hid string, userID string, state *HoldemState)
	//PlayerLeaveSuccess 玩家离开游戏成功
	PlayerLeaveSuccess(hid string, userID string)
	//PlayerSeatedSuccess 玩家坐下成功(补盲状态)
	PlayerSeatedSuccess(hid string, seat int8, userID string, te PlayType)
	//PlayerCanPayToPlay 玩家可以补盲了
	PlayerCanPayToPlay(hid string, seat int8, userID string)
	//PlayerPayToPlaySuccesss 玩家补盲成功
	PlayerPayToPlaySuccesss(hid string, seat int8, userID string)
	//PlayerReadyStandUpSuccess 玩家准备站起成功
	PlayerReadyStandUpSuccess(hid string, seat int8, userID string)
	//PlayerStandUp 玩家站起
	PlayerStandUp(hid string, seat int8, userID string, reasonCode int8)
	//PlayerKeepSeat 玩家占座(座位号)
	PlayerKeepSeat(hid string, seat int8, userID string, tm time.Duration)
	//PlayerExceedTimeSuccess 玩家延时成功
	PlayerExceedTimeSuccess(hid string, seat int8, uid string, times int8, tm time.Duration)
}
