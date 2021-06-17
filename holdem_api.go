package holdem

import (
	"time"

	"go.uber.org/zap"
)

//Start 开始游戏
func (c *Holdem) Start() {
	c.log.Debug("game start")
	c.gameStatus = GameStatusPlaying
}

//Wait 等待开始
func (c *Holdem) Wait() {
	for {
		if c.gameStatus == GameStatusNotStart {
			continue
		}
		ok := c.buttonPosition()
		if !ok {
			c.log.Debug("players are not enough, wait")
			continue
		}

		c.log.Debug("hand start")
		c.startHand()
		next, wait := c.nextGame(c.handNum)
		if next {
			//清理座位用户
			waitforbuy := false
			c.seatLock.Lock()
			for i, r := range c.players {
				r.gameInfo.ResetForNextHand()
				if r.gameInfo.chip == 0 {
					waitforbuy = true
					c.delayStandUp(i, r, c.delayStandUpTimeout)
					continue
				}
				if r.gameInfo.needStandUp {
					c.log.Debug("user stand up", zap.Int8("seat", i), zap.String("user", r.user.ID()))
					c.standUp(i, r, StandUpAction)
				}
			}
			c.seatLock.Unlock()
			c.log.Debug("hand end")
			if waitforbuy {
				if c.delayStandUpTimeout > wait-500*time.Millisecond {
					wait = c.delayStandUpTimeout + 500*time.Millisecond
				}
			}
			time.Sleep(wait)
			continue
		}
		c.gameStatus = GameStatusComplete
		//清理座位用户
		c.seatLock.Lock()
		for i, r := range c.players {
			r.gameInfo.ResetForNextHand()
			c.log.Debug("user end stand up", zap.Int8("seat", i), zap.String("user", r.user.ID()))
			c.standUp(i, r, StandUpGameEnd)
		}
		c.seatLock.Unlock()
		c.log.Debug("game end")
		return
	}
}
