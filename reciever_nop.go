package holdem

import "time"

type NopReciever struct {
}

var _ Reciever = (*NopReciever)(nil)

//ErrorOccur 接收错误
func (c *NopReciever) ErrorOccur(string, int, error) {}

//RoomerJoin 接收有人进入游戏
func (c *NopReciever) RoomerJoin(hid string, userID string) {}

//RoomerLeave 接收有人离开游戏
func (c *NopReciever) RoomerLeave(hid string, userID string) {}

//RoomerSeated 接收有人坐下
func (c *NopReciever) RoomerSeated(hid string, seat int8, userID string, te PlayType) {}

//RoomerRoomerStandUp
func (c *NopReciever) RoomerStandUp(hid string, seat int8, userID string, reaonCode int8) {}

//RoomerGetCard 接收有人收到牌（位置,牌数量,操作者)
func (c *NopReciever) RoomerGetCard(hid string, reciveSeats []int8, cardsNum int8, handInfo *StartNewHandInfo, op *Operator) {
}

//RoomerGetPublicCard 接收公共牌(牌,谁操作)
func (c *NopReciever) RoomerGetPublicCard(hid string, cards []*Card, op *Operator) {}

//RoomerGetAction 接收有人动作（位置，用户, 动作，金额(如果下注),当前要操作者)
func (c *NopReciever) RoomerGetAction(hid string, seat int8, usreID string, act ActionDef, num uint, op *Operator) {
}

//RoomerGetBuyInsurance 接收谁购买了保险的信息
func (c *NopReciever) RoomerGetBuyInsurance(hid string, seat int8, buy []*BuyInsurance, round Round) {
}

//RoomerGetShowCards 接收亮牌信息
func (c *NopReciever) RoomerGetShowCards(hid string, sc []*ShowCard) {}

//RoomerGetResult 接收牌局结果
func (c *NopReciever) RoomerGetResult(hid string, res []*Result) {}

//RoomerKeepSeat 接收有人占座(座位号)
func (c *NopReciever) RoomerKeepSeat(hid string, seat int8, userID string, tm time.Duration) {}

//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注),下一个操作者)
func (c *NopReciever) PlayerActionSuccess(hid string, seat int8, userID string, act ActionDef, num uint, op *Operator) {
}

//PlayerGetCard 玩家获得自己发到的牌（座位号,牌,发牌顺序,几张牌,下一个操作者是否是你)
func (c *NopReciever) PlayerGetCard(hid string, seat int8, userID string, cards []*Card, dealOrder []int8, num int8, handsInfo *StartNewHandInfo, op *Operator) {
}

//PlayerCanNotBuyInsurance 玩家无法购买保险(座位号,outs数量,回合)
func (c *NopReciever) PlayerCanNotBuyInsurance(hid string, seat int8, userID string, outsLen int, round Round) {
}

//PlayerCanBuyInsurance 玩家可以开始买保险(座位号,outs数量,赔率,具体outs,回合)
func (c *NopReciever) PlayerCanBuyInsurance(hid string, seat int8, userID string, outsLen int, odds float64, outs map[int8][]*UserOut, round Round) {
}

//PlayerBuyInsuranceSuccess 玩家购买保险成功（座位号，金额）
func (c *NopReciever) PlayerBuyInsuranceSuccess(hid string, seat int8, userID string, buy []*BuyInsurance) {
}

//PlayerBringInSuccess 玩家带入成功
func (c *NopReciever) PlayerBringInSuccess(hid string, seat int8, userID string, chip uint) {}

//PlayerJoinSuccess 玩家进入游戏成功
func (c *NopReciever) PlayerJoinSuccess(hid string, userID string, state *HoldemState) {}

//PlayerLeaveSuccess 玩家离开游戏成功
func (c *NopReciever) PlayerLeaveSuccess(hid string, userID string) {}

//PlayerSeatedSuccess 玩家坐下成功(补盲状态)
func (c *NopReciever) PlayerSeatedSuccess(hid string, seat int8, userID string, te PlayType) {}

//PlayerCanPayToPlay 玩家可以补盲了
func (c *NopReciever) PlayerCanPayToPlay(hid string, seat int8, userID string) {}

//PlayerPayToPlaySuccesss 玩家补盲成功
func (c *NopReciever) PlayerPayToPlaySuccesss(hid string, seat int8, userID string) {}

//PlayerReadyStandUpSuccess 玩家准备站起成功
func (c *NopReciever) PlayerReadyStandUpSuccess(hid string, seat int8, userID string) {}

//PlayerStandUp 玩家站起
func (c *NopReciever) PlayerStandUp(hid string, seat int8, userID string, reasonCode int8) {}

//PlayerKeepSeat 玩家占座(座位号)
func (c *NopReciever) PlayerKeepSeat(hid string, seat int8, userID string, tm time.Duration) {}
