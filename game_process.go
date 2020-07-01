package holdem

import (
	"time"

	"github.com/kataras/iris/core/errors"
	"go.uber.org/zap"
)

//递归投注
func (c *game) waitBet(waitPlayerSeat int8, duration time.Duration) (bool, error) {
	tm := time.NewTimer(duration)
	defer tm.Stop()
	select {
	case bet, ok := <-c.betCh:
		if !ok {
			c.log.Error("betCh error")
			return false, errors.New("betCh error ")
		}
		if waitPlayerSeat != bet.seat {
			c.log.Error("wait user wrong", zap.Int8("waitfor", waitPlayerSeat), zap.Int8("comming", bet.seat))
			return false, errors.New("wait user wrong ")
		}
		switch bet.action {
		case ActionBet:
			//通知有人下注
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerBet(c, c.players[bet.seat], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
		case ActionCall:
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerCall(c, c.players[bet.seat], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
		case ActionFold:
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerFold(c, c.players[bet.seat])
					if err != nil {
						return false, err
					}
				}
			}
		case ActionCheck:
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerCheck(c, c.players[bet.seat])
					if err != nil {
						return false, err
					}
				}
			}
		case ActionRaise:
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerRaise(c, c.players[bet.seat], bet.bet)
					if err != nil {
						return false, err
					}
				}
			}
		case ActionAllIn:
			for i, v := range c.players {
				if i != bet.seat {
					err := v.listener.PlayerAllIn(c, c.players[bet.seat], bet.bet)
					if err != nil {
						return false, err
					}
				}
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
			if i != waitPlayerSeat {
				err := v.listener.PlayerFold(c, c.players[waitPlayerSeat])
				if err != nil {
					return false, err
				}
			}
		}
	}
	seat, _ := c.findNextBeter(waitPlayerSeat)
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
			mp[v.seat], err = v.caculateMaxHandValue(c.publicCards)
			if err != nil {
				return -1, err
			}
		}
		mp = GetMaxHandValueFromTaggedHandValues(mp)
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
