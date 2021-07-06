package holdem

import (
	"sync/atomic"
)

//Start 开始游戏
func (c *Holdem) Start() {
	if atomic.LoadInt32(&c.gameStartedLock) == 0 {
		atomic.StoreInt32(&c.gameStartedLock, int32(GameStatusStarted))
		c.gameStatusCh <- GameStatusStarted
	}
}

//Cancel 提前取消
func (c *Holdem) Cancel() {
	if atomic.LoadInt32(&c.gameStartedLock) == 0 {
		c.gameStatusCh <- GameStatusCancel
		atomic.StoreInt32(&c.gameStartedLock, int32(GameStatusCancel))
	} else {
		c.log.Warn("can not cancel a started game")
	}
}

//State 状态
func (c *Holdem) State(rs ...*Agent) *HoldemState {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	return c.information(rs...)
}

//ChangeBetConfig 修改下注配置（小盲/前注)
func (c *Holdem) ChangeBetConfig(sb uint, ante ...uint) {
	c.nextSb = int(sb)
	if len(ante) > 0 {
		c.nextAnte = int(ante[0])
	}
}

//ForceStandUp 强制让人站起
func (c *Holdem) ForceStandUp(id ...string) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	rm := make(map[string]bool)
	for _, d := range id {
		rm[d] = true
	}
	for seat, r := range c.players {
		if _, ok := rm[r.id]; ok {
			r.gameInfo.needStandUpReason = StandUpGameForce
			if r.gameInfo.status == ActionDefNone {
				c.standUp(seat, r, StandUpGameForce)
				return
			}
		}
	}
}

//ForcePlayerStandUp
func (c *Holdem) ForcePlayerStandUp(count uint8) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	buIdx := c.bbSeat + 1
	if buIdx > c.seatCount {
		buIdx = 1
	}
	num := count
	var i int8
	//从大盲位开始站起
	for i = 0; i < c.seatCount; i++ {
		seat := ((i + buIdx - 1) % c.seatCount) + 1
		r, ok := c.players[seat]
		if ok {
			r.gameInfo.needStandUpReason = StandUpGameExchange
			if r.gameInfo.status == ActionDefNone {
				c.standUp(seat, r, StandUpGameForce)
			}
			num--
			if num == 0 {
				return
			}
		}
	}
}
