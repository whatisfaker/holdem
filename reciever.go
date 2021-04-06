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
	//RoomerGetBuyInsurance 接收谁购买了保险的信息
	RoomerGetBuyInsurance(seat int8, buy []*BuyInsurance, round Round)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards([]*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult([]*Result)
	//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注))
	PlayerActionSuccess(int8, int8, ActionDef, int)
	//PlayerGetCard 玩家获得自己发到的牌（座位号,牌,发牌顺序,几张牌)
	PlayerGetCard(int8, []*Card, []int8, int8)
	//PlayerCanNotBuyInsurance 玩家无法购买保险(座位号,outs数量,回合)
	PlayerCanNotBuyInsurance(seat int8, outsLen int, round Round)
	//PlayerCanBuyInsurance 玩家可以开始买保险(座位号,outs数量,赔率,具体outs,回合)
	PlayerCanBuyInsurance(seat int8, outsLen int, odds float64, outs map[int8][]*UserOut, round Round)
	//PlayerBuyInsuranceSuccess 玩家购买保险成功（座位号，金额）
	PlayerBuyInsuranceSuccess(seat int8, buy []*BuyInsurance)
	//PlayerCanBet 玩家可以开始下注(剩下筹码,本手已下注,本轮下注数量, 本轮的筹码数量, 最小下注额度)
	PlayerCanBet(seat int8, chip int, handBet int, roundBet int, curBet int, minBet int, round Round)
	//PlayerBringInSuccess 玩家带入成功
	PlayerBringInSuccess(seat int8, chip int)
	//PlayerSeatedSuccess 玩家坐下成功
	PlayerSeatedSuccess(int8)
	//PlayerReadyStandUpSuccess 玩家准备站起成功
	PlayerReadyStandUpSuccess(int8)
	//PlayerStandUp 玩家站起
	PlayerStandUp(int8)
}
