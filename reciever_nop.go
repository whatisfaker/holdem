package holdem

import "time"

type NopReciever struct {
}

var _ Reciever = (*NopReciever)(nil)

//ErrorOccur 接收错误
func (c *NopReciever) ErrorOccur(int, error) {}

//RoomerGameInformation 游戏信息
func (c *NopReciever) RoomerGameInformation(*HoldemState) {}

//RoomerJoin 接收有人进入游戏{}
func (c *NopReciever) RoomerJoin(string) {}

//RoomerLeave 接收有人离开游戏
func (c *NopReciever) RoomerLeave(string) {}

//RoomerSeated 接收有人坐下{}
func (c *NopReciever) RoomerSeated(int8, string, PlayType) {}

//RoomerRoomerStandUp{}
func (c *NopReciever) RoomerStandUp(int8, string, int8) {}

//RoomerGetCard 接收有人收到牌（位置,牌数量,操作者){}
func (c *NopReciever) RoomerGetCard([]int8, int8, *StartNewHandInfo, *Operator) {}

//RoomerGetPublicCard 接收公共牌(牌,谁操作,是否是你操作){}
func (c *NopReciever) RoomerGetPublicCard([]*Card, *Operator, bool) {}

//RoomerGetAction 接收有人动作（按钮位, 位置，动作，金额(如果下注), 是否是你){}
func (c *NopReciever) RoomerGetAction(int8, int8, ActionDef, uint, *Operator, bool) {}

//RoomerGetBuyInsurance 接收谁购买了保险的信息{}
func (c *NopReciever) RoomerGetBuyInsurance(seat int8, buy []*BuyInsurance, round Round) {}

//RoomerGetShowCards 接收亮牌信息{}
func (c *NopReciever) RoomerGetShowCards([]*ShowCard) {}

//RoomerGetResult 接收牌局结果{}
func (c *NopReciever) RoomerGetResult([]*Result) {}

//RoomerKeepSeat 接收有人占座(座位号){}
func (c *NopReciever) RoomerKeepSeat(int8, time.Duration) {}

//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注),下一个操作者){}
func (c *NopReciever) PlayerActionSuccess(int8, int8, ActionDef, uint, *Operator) {}

//PlayerGetCard 玩家获得自己发到的牌（座位号,牌,发牌顺序,几张牌,下一个操作者是否是你){}
func (c *NopReciever) PlayerGetCard(int8, []*Card, []int8, int8, *StartNewHandInfo, *Operator, bool) {
}

//PlayerCanNotBuyInsurance 玩家无法购买保险(座位号,outs数量,回合){}
func (c *NopReciever) PlayerCanNotBuyInsurance(seat int8, outsLen int, round Round) {}

//PlayerCanBuyInsurance 玩家可以开始买保险(座位号,outs数量,赔率,具体outs,回合){}
func (c *NopReciever) PlayerCanBuyInsurance(seat int8, outsLen int, odds float64, outs map[int8][]*UserOut, round Round) {
}

//PlayerBuyInsuranceSuccess 玩家购买保险成功（座位号，金额）{}
func (c *NopReciever) PlayerBuyInsuranceSuccess(seat int8, buy []*BuyInsurance) {}

//PlayerBringInSuccess 玩家带入成功{}
func (c *NopReciever) PlayerBringInSuccess(seat int8, chip uint) {}

//PlayerJoinSuccess 玩家进入游戏成功{}
func (c *NopReciever) PlayerJoinSuccess(string, *HoldemState) {}

//PlayerLeaveSuccess 玩家离开游戏成功{}
func (c *NopReciever) PlayerLeaveSuccess(string) {}

//PlayerSeatedSuccess 玩家坐下成功(补盲状态){}
func (c *NopReciever) PlayerSeatedSuccess(int8, PlayType) {}

//PlayerCanPayToPlay 玩家可以补盲了{}
func (c *NopReciever) PlayerCanPayToPlay(int8) {}

//PlayerPayToPlaySuccesss 玩家补盲成功{}
func (c *NopReciever) PlayerPayToPlaySuccesss(int8) {}

//PlayerReadyStandUpSuccess 玩家准备站起成功{}
func (c *NopReciever) PlayerReadyStandUpSuccess(int8) {}

//PlayerStandUp 玩家站起{}
func (c *NopReciever) PlayerStandUp(int8, int8) {}

//PlayerKeepSeat 玩家占座(座位号){}
func (c *NopReciever) PlayerKeepSeat(int8, time.Duration) {}
