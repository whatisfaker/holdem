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
	*holdem.NopReciever
	ch  chan<- *ServerAction
	id  string
	log *zap.Logger
}

var _ holdem.Reciever = (*rec)(nil)

//RoomerSeated 接收有人坐下
func (c *rec) RoomerSeated(seat int8, u string, payToPlay holdem.PlayType) {
	c.ch <- &ServerAction{
		Action: SASeated,
		Seat:   seat,
	}
}

func (c *rec) RoomerGameInformation(h *holdem.HoldemState) {
	seats := h.EmptySeats
	b, _ := json.Marshal(seats)
	c.ch <- &ServerAction{
		Action:  SAGame,
		Payload: b,
	}
}

func (c *rec) PlayerJoinSuccess(userinfo string, h *holdem.HoldemState) {
	seats := h.EmptySeats
	b, _ := json.Marshal(seats)
	c.ch <- &ServerAction{
		Action:  SAGame,
		Payload: b,
	}
}

//RoomerGetCard 接收有人收到牌（位置,牌数量)
func (c *rec) RoomerGetCard(a []int8, num int8, info *holdem.StartNewHandInfo, op *holdem.Operator) {
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
func (c *rec) RoomerGetPublicCard(cds []*holdem.Card, op *holdem.Operator) {
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAGetPCards,
		Payload: b,
	}
	if op != nil && op.ID == c.id {
		c.ch <- &ServerAction{
			Action: SACanBet,
			Seat:   op.SeatNumber,
			BetInfo: &BetInfo{
				Chip:       op.Chip,
				HandBet:    op.HandBet,
				RoundBet:   op.RoundBet,
				CurrentBet: op.CurrentTableBet,
				MinRaise:   op.MinRaise,
			},
		}
	}
}

//RoomerGetShowCards 接收亮牌信息
func (c *rec) RoomerGetShowCards(cds []*holdem.ShowCard) {
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAShowCards,
		Payload: b,
	}
}

//RoomerGetAction 接收有人动作（位置，动作，金额(如果下注))
func (c *rec) RoomerGetAction(seat int8, id string, action holdem.ActionDef, num uint, op *holdem.Operator) {
	c.ch <- &ServerAction{
		Action:  SAAction,
		Action2: action,
		Seat:    seat,
		Num:     num,
	}
	if op != nil && op.ID == c.id {
		c.ch <- &ServerAction{
			Action: SACanBet,
			Seat:   op.SeatNumber,
			BetInfo: &BetInfo{
				Chip:       op.Chip,
				HandBet:    op.HandBet,
				RoundBet:   op.RoundBet,
				CurrentBet: op.CurrentTableBet,
				MinRaise:   op.MinRaise,
			},
		}
	}
}

//RoomerGetResult 接收牌局结果
func (c *rec) RoomerGetResult(rs []*holdem.Result) {
	b, _ := json.Marshal(rs)
	c.ch <- &ServerAction{
		Action:  SAResult,
		Payload: b,
	}
}

//PlayerGetCard 玩家获得自己发到的牌
func (c *rec) PlayerGetCard(seat int8, id string, cds []*holdem.Card, seats []int8, cnt int8, info *holdem.StartNewHandInfo, op *holdem.Operator) {
	mp := make(map[string]interface{})
	mp["cards"] = cds
	mp["order"] = seats
	mp["num"] = cnt
	b, _ := json.Marshal(cds)
	c.ch <- &ServerAction{
		Action:  SAGetMyCards,
		Payload: b,
	}
	if op != nil && op.ID == c.id {
		c.ch <- &ServerAction{
			Action: SACanBet,
			Seat:   op.SeatNumber,
			BetInfo: &BetInfo{
				Chip:       op.Chip,
				HandBet:    op.HandBet,
				RoundBet:   op.RoundBet,
				CurrentBet: op.CurrentTableBet,
				MinRaise:   op.MinRaise,
			},
		}
	}
}

func (c *rec) ErrorOccur(code int, err error) {
	//c.log.Error("error occur", zap.Error(err))
	c.ch <- &ServerAction{
		Action:  SAError,
		Num:     uint(code),
		Payload: []byte(err.Error()),
	}
}

func (c *rec) PlayerBringInSuccess(seat int8, id string, chip uint) {
	c.ch <- &ServerAction{
		Action: SABringInSuccess,
		Seat:   seat,
		Num:    chip,
	}
}

func (c *rec) PlayerSeatedSuccess(seat int8, id string, payToPlay holdem.PlayType) {
	c.ch <- &ServerAction{
		Action: SASelfSeated,
		Seat:   seat,
	}
}

func (c *rec) PlayerReadyStandUpSuccess(seat int8, id string) {
	c.ch <- &ServerAction{
		Action: SAReadyStandUp,
		Seat:   seat,
	}
}

//PlayerActionSuccess 玩家动作成功（按钮位, 位置，动作，金额(如果下注))
func (c *rec) PlayerActionSuccess(s int8, userID string, act holdem.ActionDef, num uint, h *holdem.Operator) {
	c.ch <- &ServerAction{
		Action:  SAMyAction,
		Action2: act,
		Num:     num,
		Seat:    s,
	}
}

func (c *rec) PlayerStandUp(seat int8, userID string, reason int8) {
	c.ch <- &ServerAction{
		Action: SAStandUp,
		Seat:   seat,
	}
}
