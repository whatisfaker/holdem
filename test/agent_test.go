package holdem

import (
	"fmt"
	"math/rand"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
)

type rec struct {
	id  string
	log *zap.Logger
}

var _ holdem.Reciever = (*rec)(nil)

func (c *rec) ID() string {
	return c.id
}

//RoomerSeated 接收有人坐下
func (c *rec) RoomerSeated(seat int8, u holdem.UserInfo) {
	c.log.Info("RoomerSeated", zap.String("id", c.id), zap.Int8("seat", seat), zap.String("who", u.Name()))
}

//RoomerRoomerStandUp
func (c *rec) RoomerStandUp(int8, holdem.UserInfo) {
	c.log.Info("RoomerStandUp")
}

//RoomerGetCard 接收有人收到牌（位置,牌数量)
func (c *rec) RoomerGetCard(a []int8, b int8) {
	c.log.Info("RoomerGetCard", zap.String("id", c.id), zap.Int8s("seats", a), zap.Int8("count", b))
}

//RoomerGetPublicCard 接收公共牌
func (c *rec) RoomerGetPublicCard([]*holdem.Card) {
	c.log.Info("RoomerSeated")
}

//RoomerGetShowCards 接收亮牌信息
func (c *rec) RoomerGetShowCards([]*holdem.ShowCard) {
	c.log.Info("RoomerGetShowCards", zap.String("id", c.id))
}

//RoomerGetAction 接收有人动作（位置，动作，金额(如果下注))
func (c *rec) RoomerGetAction(seat int8, action holdem.ActionDef, num ...int) {
	c.log.Info("RoomerGetAction", zap.String("id", c.id), zap.Int8("seat", seat), zap.String("action", action.String()), zap.Ints("num", num))
}

//RoomerGetResult 接收牌局结果
func (c *rec) RoomerGetResult(rs []*holdem.Result) {
	c.log.Info("RoomerGetResult", zap.String("id", c.id), zap.Any("result", rs))
}

//PlayerGetCard 玩家获得自己发到的牌
func (c *rec) PlayerGetCard(cds []*holdem.Card) {
	c.log.Info("PlayerGetCard", zap.String("id", c.id), zap.Any("cards", cds))
}

func (c *rec) ErrorOccur(err error) {
	c.log.Error("error occur", zap.Error(err))
}

//PlayerCanBet 玩家可以开始下注
func (c *rec) PlayerCanBet() {
	c.log.Info("PlayerCanBet", zap.String("id", c.id))
}

type player struct {
	name   string
	avatar string
}

var _ holdem.UserInfo = (*player)(nil)

func (c *player) Name() string {
	return c.name
}

func (c *player) Avatar() string {
	return c.avatar
}

func (c *player) Info() map[string]string {
	return nil
}

func newTestAgent(l *zap.Logger) *holdem.Agent {
	id := rand.Intn(100)
	return holdem.NewAgent(&rec{
		id:  fmt.Sprint(id),
		log: l,
	}, &player{
		name:   fmt.Sprintf("na-%d", id),
		avatar: "",
	})
}
