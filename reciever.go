package holdem

type Reciever interface {
	//ErrorOccur 接收错误
	ErrorOccur(int, error)
	//RoomerGameInformation 接收游戏信息
	RoomerGameInformation(*Holdem)
	//RoomerSeated 接收有人坐下
	RoomerSeated(int8, UserInfo)
	//RoomerRoomerStandUp
	RoomerStandUp(int8, UserInfo)
	//RoomerGetCard 接收有人收到牌（位置,牌数量)
	RoomerGetCard([]int8, int8)
	//RoomerGetPublicCard 接收公共牌
	RoomerGetPublicCard([]*Card)
	//RoomerGetAction 接收有人动作（按钮位, 位置，动作，金额(如果下注))
	RoomerGetAction(int8, int8, ActionDef, int)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards([]*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult([]*Result)
	//PlayerGetCard 玩家获得自己发到的牌
	PlayerGetCard(int8, []*Card, []int8, int8)
	//PlayerCanBet 玩家可以开始下注(剩下筹码,本手已下注,本轮下注数量, 本轮的筹码数量, 最小下注额度)
	PlayerCanBet(seat int8, chip int, handBet int, roundBet int, curBet int, minBet int, round Round)
	//PlayerBringInSuccess 玩家带入成功
	PlayerBringInSuccess(seat int8, chip int)
	//PlayerSeated 玩家坐下
	PlayerSeated(int8)
	//PlayerStandUp 玩家站起
	PlayerStandUp(int8)
}
