package holdem

import (
	"time"

	"github.com/kataras/iris/core/errors"
	"go.uber.org/zap"
)

//递归投注
func (c *game) waitBet(waitPlayerNum int8, duration time.Duration) (bool, error) {
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
					err := v.listener.PlayerBet(c, c.players[bet.num], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerBet(c, c.players[bet.num], bet.bet)
			if err != nil {
				return false, err
			}
		case ActionCall:
			for i, v := range c.players {
				if i != bet.num {
					err := v.listener.PlayerCall(c, c.players[bet.num], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerCall(c, c.players[bet.num], bet.bet)
			if err != nil {
				return false, err
			}
		case ActionFold:
			for i, v := range c.players {
				if i != bet.num {
					err := v.listener.PlayerFold(c, c.players[bet.num])
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerFold(c, c.players[bet.num])
			if err != nil {
				return false, err
			}
		case ActionCheck:
			for i, v := range c.players {
				if i != bet.num {
					err := v.listener.PlayerCheck(c, c.players[bet.num])
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerCheck(c, c.players[bet.num])
			if err != nil {
				return false, err
			}
		case ActionRaise:
			for i, v := range c.players {
				if i != bet.num {
					err := v.listener.PlayerRaise(c, c.players[bet.num], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerRaise(c, c.players[bet.num], bet.bet)
			if err != nil {
				return false, err
			}
		case ActionAllIn:
			for i, v := range c.players {
				if i != bet.num {
					err := v.listener.PlayerAllIn(c, c.players[bet.num], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
			err := c.listener.PlayerAllIn(c, c.players[bet.num], bet.bet)
			if err != nil {
				return false, err
			}
		}
	case val, ok := <-c.pauseCh:
		if !ok {
			c.log.Error("pause error")
			return false, errors.New("betCh error ")
		}
		if val == 1 {
			<-c.pauseCh
		}
	case <-tm.C:
		for i, v := range c.players {
			if i != waitPlayerNum {
				err := v.listener.PlayerFold(c, c.players[waitPlayerNum])
				if err != nil {
					return false, err
				}
			}
		}
		err := c.listener.PlayerFold(c, c.players[waitPlayerNum])
		if err != nil {
			return false, err
		}
	}
	seat, _ := c.findNextBeter(waitPlayerNum)
	if seat < 0 {
		return false, nil
	}
	err := c.players[seat].listener.BetStart(c)
	if err != nil {
		return false, err
	}
	return c.waitBet(seat, 30*time.Second)
}

func (c *game) findNextBeter(seat int8) (int8, error) {
	leftPlayer := make([]*player, 0)
	var i int8
	var nextSeat int8 = -1
	loopSeat := seat
	for i = 0; i < c.playerCount-1; i++ {
		loopSeat = c.nextSeat(loopSeat)
		player := c.players[seat]
		if player.status != PlayerStatusFold {
			if nextSeat == -1 && i != seat {
				nextSeat = seat
			}
			leftPlayer = append(leftPlayer, player)
		}
	}
	//剩下一个直接结算
	if len(leftPlayer) == 1 {
		err := leftPlayer[0].win(c.pod)
		if err != nil {
			return -1, err
		}
		c.settlement()
		return nextSeat, nil
	}
	if c.step == StepRiverRound {
		mp := make(map[int8]*HandValue)
		var err error
		for _, v := range leftPlayer {
			mp[v.number], err = v.caculateMaxHandValue(c.publicCards)
			if err != nil {
				return -1, err
			}
		}
		mp = GetMaxHandValue(mp)
		c.dealWinner(mp)
		c.settlement()
		return nextSeat, nil
	}

	return nextSeat, nil
}

func (c *game) dealWinner(map[int8]*HandValue) {

}

func (c *game) settlement() {

}
