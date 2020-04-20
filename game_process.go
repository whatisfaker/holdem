package holdem

import (
	"time"

	"github.com/kataras/iris/core/errors"
	"go.uber.org/zap"
)

//递归投注
func (c *Game) waitBet(waitPlayerNum int8, duration time.Duration) (bool, error) {
	tm := time.NewTimer(duration)
	defer tm.Stop()
	select {
	case bet, ok := <-c.betCh:
		if !ok {
			c.log.Error("betCh error")
			return false, errors.New("betCh error ")
		}
		if waitPlayerNum != bet.num {
			c.log.Error("wait user wrong", zap.Int8("waitfor", waitPlayerNum), zap.Int8("comming", bet.num))
			return false, errors.New("wait user wrong ")
		}
		switch bet.action {
		case ActionBet:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.Bet(c, c.players[bet.num], bet.bet)
				}
			}
		case ActionCall:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.Call(c, c.players[bet.num])
				}
			}
		case ActionFold:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.Fold(c, c.players[bet.num])
				}
			}
		case ActionCheck:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.Check(c, c.players[bet.num])
				}
			}
		case ActionRaise:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.Raise(c, c.players[bet.num], bet.bet)
				}
			}
		case ActionAllIn:
			for i, v := range c.players {
				if i != bet.num {
					v.listener.AllIn(c, c.players[bet.num], bet.bet)
				}
			}
		}
	case <-tm.C:
		for i, v := range c.players {
			if i != waitPlayerNum {
				v.listener.Fold(c, c.players[waitPlayerNum])
			}
		}
	}
	seat := c.findNextBeter(waitPlayerNum)
	if seat < 0 {
		return false, nil
	}
	c.players[seat].listener.BetStart(c)
	return c.waitBet(seat, 30*time.Second)
}

func (c *Game) findNextBeter(seat int8) int8 {
	return -1
}
