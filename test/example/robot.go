package example

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
)

type RA int8

const (
	RAJoin RA = iota + 1
	RAInfo
	RABringIn
	RASeat
	RABet
	RAStandUp
)

type SA int8

const (
	SAGame SA = iota + 1
	SAStandUp
	SASeated
	SAGetCards
	SAGetMyCards
	SASelfSeated
	SAReadyStandUp
	SAGetPCards
	SAShowCards
	SACanBet
	SAAction
	SAMyAction
	SAResult
	SAError
	SABringInSuccess
)

type Robot struct {
	seated bool
	nochip bool
	inCh   chan *ServerAction
	outCh  chan *RobotAction
	seats  []int8
	log    *zap.Logger
}

type RobotAction struct {
	Action RA
	Num    int
	Bet    *holdem.Bet
}

func NewRobot(log *zap.Logger) *Robot {
	r := &Robot{
		log:    log,
		nochip: true,
	}
	r.inCh = make(chan *ServerAction, 10)
	r.outCh = make(chan *RobotAction, 10)
	go r.read()
	return r
}

func (c *Robot) OutCh() <-chan *RobotAction {
	return c.outCh
}

func (c *Robot) InCh() chan<- *ServerAction {
	return c.inCh
}

func (c *Robot) Start() {
	c.outCh <- &RobotAction{
		Action: RAJoin,
	}
}

func (c *Robot) read() {
	defer close(c.outCh)
	defer close(c.inCh)
	for in := range c.inCh {
		switch in.Action {
		case SAError:
			if in.Num == holdem.ErrCodeSeatTaken {
				c.log.Warn("error ", zap.String("error", string(in.Payload)))
				time.AfterFunc(time.Second, func() {
					if len(c.seats) > 0 {
						seat := c.seats[rand.Intn(len(c.seats))]
						c.outCh <- &RobotAction{
							Action: RASeat,
							Num:    int(seat),
						}
					}
				})
			} else {
				c.log.Error("error ", zap.String("error", string(in.Payload)))
			}
		case SABringInSuccess:
			if len(c.seats) > 0 {
				seat := c.seats[rand.Intn(len(c.seats))]
				c.log.Debug("SABringInSuccess", zap.Int8("trysit", seat))
				//fmt.Println("bringin", c.seats, seat)
				c.outCh <- &RobotAction{
					Action: RASeat,
					Num:    int(seat),
				}
			}
		case SAGame:
			c.log.Debug("SAGame", zap.String("seats", string(in.Payload)))
			var seats []int8
			_ = json.Unmarshal(in.Payload, &seats)
			c.seats = seats
			if len(c.seats) > 0 {
				if c.nochip {
					c.outCh <- &RobotAction{
						Action: RABringIn,
						Num:    10000,
					}
				}
				// } else {

				// 	seat := c.seats[rand.Intn(len(c.seats))]
				// 	//fmt.Println("haschip", c.seats, seat)
				// 	c.log.Debug("SAGame", zap.Int8("re-trysit", seat))
				// 	c.outCh <- &RobotAction{
				// 		Action: RASeat,
				// 		Num:    int(seat),
				// 	}
				// }
			}
		case SAStandUp:
			c.seated = false
			if int8(in.Num) == holdem.StandUpNoChip {
				c.nochip = true
			} else {
				c.nochip = false
			}
			//站起来了
			c.log.Debug("SAStandUp", zap.Int8("seat", in.Seat))
			time.AfterFunc(time.Second, func() {
				c.outCh <- &RobotAction{
					Action: RAInfo,
				}
			})
		case SAReadyStandUp:
			//准备站起
			c.log.Debug("SAReadyStandUp", zap.Int8("seat", in.Seat))
		case SASeated:
			c.log.Debug("SASeated", zap.Int8("seat", in.Seat))
			seat := in.Seat
			j := 0
			for _, v := range c.seats {
				if v != int8(seat) {
					c.seats[j] = v
					j++
				}
			}
			c.seats = c.seats[:j]
		case SASelfSeated:
			c.seated = true
			c.log.Debug("SASelfSeated", zap.Int8("seat", in.Seat))
		case SACanBet:
			bet := c.MyAction(in.BetInfo)
			c.log.Debug("SACanBet", zap.String("action", bet.Action.String()), zap.Any("get", in.BetInfo), zap.Int8("seat", in.Seat), zap.Any("bet", bet))
			c.outCh <- &RobotAction{
				Action: RABet,
				Bet:    bet,
			}
		case SAGetMyCards:
			c.log.Debug("SAGetMyCards", zap.String("info", string(in.Payload)))
		case SAGetCards:
			c.log.Debug("SAGetCards", zap.String("info", string(in.Payload)))
		case SAGetPCards:
			c.log.Debug("SAGetPCards", zap.String("info", string(in.Payload)))
		case SAShowCards:
			c.log.Debug("SAShowCards", zap.String("info", string(in.Payload)))
		case SAMyAction:
			if in.Action2 == holdem.ActionDefFold {
				c.outCh <- &RobotAction{
					Action: RAStandUp,
				}
			}
		case SAAction:
			c.log.Debug("SAAction", zap.Int8("seat", in.Seat), zap.String("action", in.Action2.String()), zap.Int("num", in.Num))
		case SAResult:
			results := make([]*holdem.Result, 0)
			_ = json.Unmarshal(in.Payload, &results)
			c.log.Debug("SAResult", zap.String("result", string(in.Payload)))
			if c.seated {
				c.outCh <- &RobotAction{
					Action: RAStandUp,
				}
			}
		}
	}

}

func (c *Robot) MyAction(bet *BetInfo) *holdem.Bet {
	//第一个人/或者前面没有人下注
	actions := make([]Choice, 0)
	//c.log.Error("get bet", zap.Any("bet", bet))
	if bet.CurrentBet == 0 {
		if bet.Chip > bet.MinRaise {
			actions = append(actions,
				NewChoice(holdem.ActionDefFold, 40),
				NewChoice(holdem.ActionDefBet, 35),
				NewChoice(holdem.ActionDefCheck, 20),
				NewChoice(holdem.ActionDefAllIn, 5))
		} else {
			actions = append(actions,
				NewChoice(holdem.ActionDefFold, 50),
				NewChoice(holdem.ActionDefCheck, 40),
				NewChoice(holdem.ActionDefAllIn, 10))
		}
	} else {
		//筹码大于当前下注
		if bet.Chip > bet.CurrentBet-bet.RoundBet+bet.MinRaise {
			actions = append(actions,
				NewChoice(holdem.ActionDefFold, 30),
				NewChoice(holdem.ActionDefCall, 40),
				NewChoice(holdem.ActionDefRaise, 25),
				NewChoice(holdem.ActionDefAllIn, 5))
		} else if bet.Chip > bet.CurrentBet-bet.RoundBet {
			actions = append(actions,
				NewChoice(holdem.ActionDefFold, 50),
				NewChoice(holdem.ActionDefCall, 45),
				NewChoice(holdem.ActionDefAllIn, 5))
		} else {
			actions = append(actions,
				NewChoice(holdem.ActionDefFold, 80),
				NewChoice(holdem.ActionDefAllIn, 20))
		}
	}
	chooser, _ := NewChooser(actions...)
	act := chooser.Pick().(holdem.ActionDef)
	switch act {
	default:
		return &holdem.Bet{
			Action: holdem.ActionDefFold,
		}
	case holdem.ActionDefCheck:
		return &holdem.Bet{
			Action: holdem.ActionDefCheck,
		}
	case holdem.ActionDefBet:
		return &holdem.Bet{
			Action: holdem.ActionDefBet,
			Num:    rand.Intn(bet.Chip-bet.MinRaise) + bet.MinRaise,
		}
	case holdem.ActionDefCall:
		return &holdem.Bet{
			Action: holdem.ActionDefCall,
			Num:    bet.CurrentBet - bet.RoundBet,
		}
	case holdem.ActionDefRaise:
		return &holdem.Bet{
			Action: holdem.ActionDefRaise,
			Num:    rand.Intn(bet.Chip-(bet.CurrentBet-bet.RoundBet+bet.MinRaise)) + bet.CurrentBet - bet.RoundBet + bet.MinRaise,
		}
	case holdem.ActionDefAllIn:
		return &holdem.Bet{
			Action: holdem.ActionDefAllIn,
			Num:    bet.Chip,
		}
	}
}
