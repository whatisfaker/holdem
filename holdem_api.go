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
func (c *Holdem) State() *HoldemState {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	return c.information()
}

//ChangeBetConfig 修改下注配置（小盲/前注)
func (c *Holdem) ChangeBetConfig(sb uint, ante ...uint) {
	c.nextSb = int(sb)
	if len(ante) > 0 {
		c.nextAnte = int(ante[0])
	}
}

//ForceStandUp 强制让人站起
func (c *Holdem) ForceStandUp(seat int8) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	if r, ok := c.players[seat]; ok {
		r.gameInfo.needStandUpReason = StandUpGameForce
		if r.gameInfo.status == ActionDefNone {
			c.standUp(seat, r, StandUpGameForce)
			return
		}
	}
}
