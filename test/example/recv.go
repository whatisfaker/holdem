package example

import (
	"encoding/json"
	"io"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func parseLevelString(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	}
	return zapcore.InfoLevel
}
func NewLogger(level string, wr io.Writer) *zap.Logger {
	logLevel := zap.NewAtomicLevel()
	logLevel.SetLevel(parseLevelString(level))
	w := zapcore.AddSync(wr)
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		w,
		logLevel,
	)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

type rec struct {
	ch  chan<- *ServerAction
	id  string
	log *zap.Logger
}

var _ holdem.Reciever = (*rec)(nil)

func (c *rec) ID() string {
	return c.id
}

//RoomerSeated 接收有人坐下
func (c *rec) RoomerSeated(seat int8, u holdem.UserInfo) {
	c.log.Debug("RoomerSeated", zap.String("id", c.id), zap.Int8("seat", seat), zap.String("who", u.Name()))

	c.ch <- &ServerAction{
		Action: SASeated,
		Seat:   seat,
	}
}

func (c *rec) RoomerGameInformation(h *holdem.Holdem) {
	c.log.Debug("RoomerGameInformation")
	_, _, _, _, seatCount, players, _ := h.Information()
	seats := make([]int, 0)
	mp := make(map[int8]bool)
	for _, v := range players {
		mp[v.SeatNumber] = true
	}
	for i := 1; i < int(seatCount)+1; i++ {
		if _, ok := mp[int8(i)]; !ok {
			seats = append(seats, i)
		}
	}
	b, _ := json.Marshal(seats)
	c.ch <- &ServerAction{
		Action:  SAGame,
		Payload: b,
	}
}

//RoomerRoomerStandUp
func (c *rec) RoomerStandUp(int8, holdem.UserInfo) {
	c.log.Debug("RoomerStandUp")
}

//RoomerGetCard 接收有人收到牌（位置,牌数量)
func (c *rec) RoomerGetCard(a []int8, num int8) {
	c.log.Debug("RoomerGetCard", zap.String("id", c.id), zap.Int8s("seats", a), zap.Int8("count", num))
	mp := make(map[string]interface{})
	mp["order"] = a
	mp["num"] = num
	b, _ := json.Marshal(mp)
	c.ch <- &ServerAction{
		Action:  SAGetCards,
		Payload: b,
	}
}

//RoomerGetPublicCard 接收公共牌
func (c *rec) RoomerGetPublicCard(cds []*holdem.Card) {
	c.log.Debug("RoomerGetPublicCard")
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAGetPCards,
		Payload: b,
	}
}

//RoomerGetShowCards 接收亮牌信息
func (c *rec) RoomerGetShowCards(cds []*holdem.ShowCard) {
	c.log.Debug("RoomerGetShowCards", zap.String("id", c.id))
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAShowCards,
		Payload: b,
	}
}

//RoomerGetAction 接收有人动作（位置，动作，金额(如果下注))
func (c *rec) RoomerGetAction(button int8, seat int8, action holdem.ActionDef, num int) {
	c.log.Debug("RoomerGetAction", zap.String("id", c.id), zap.Int8("button", button), zap.Int8("seat", seat), zap.String("action", action.String()), zap.Int("num", num))
	c.ch <- &ServerAction{
		Action:  SAAction,
		Action2: action,
		Seat:    seat,
		Num:     num,
	}
}

//RoomerGetResult 接收牌局结果
func (c *rec) RoomerGetResult(rs []*holdem.Result) {
	c.log.Debug("RoomerGetResult", zap.String("id", c.id), zap.Any("result", rs))
	b, _ := json.Marshal(rs)
	c.ch <- &ServerAction{
		Action:  SAResult,
		Payload: b,
	}
}

//PlayerGetCard 玩家获得自己发到的牌
func (c *rec) PlayerGetCard(seat int8, cds []*holdem.Card, seats []int8, cnt int8) {
	c.log.Debug("PlayerGetCard", zap.String("id", c.id), zap.Any("cards", cds), zap.Int8("seat", seat), zap.Int8s("orders", seats), zap.Int8("count", cnt))
	mp := make(map[string]interface{})
	mp["cards"] = cds
	mp["order"] = seats
	mp["num"] = cnt
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAGetMyCards,
		Payload: b,
	}
}

func (c *rec) ErrorOccur(code int, err error) {
	c.log.Error("error occur", zap.Error(err))
	c.ch <- &ServerAction{
		Action:  SAError,
		Num:     code,
		Payload: []byte(err.Error()),
	}
}

//PlayerCanBet 玩家可以开始下注
func (c *rec) PlayerCanBet(seat int8, chip int, handBet int, roundBet int, curBet int, minBet int, round int8) {
	c.log.Debug("PlayerCanBet", zap.String("id", c.id))
	c.ch <- &ServerAction{
		Action: SACanBet,
		Seat:   seat,
		BetInfo: &BetInfo{
			Chip:       chip,
			HandBet:    handBet,
			RoundBet:   roundBet,
			CurrentBet: curBet,
			MinRaise:   minBet,
		},
	}
}

func (c *rec) PlayerBringInSuccess(seat int8, chip int) {
	c.log.Debug("PlayerBringInSuccess", zap.String("id", c.id))
	c.ch <- &ServerAction{
		Action: SABringInSuccess,
		Seat:   seat,
		Num:    chip,
	}
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
