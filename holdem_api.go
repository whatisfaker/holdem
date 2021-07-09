package holdem

import (
	"sync/atomic"
)

//Start 开始游戏
func (c *Holdem) Start() {
	if atomic.LoadInt32(&c.gameStartedLock) == 0 {
		c.statusChange(GameStatusWaitHandStart)
		c.gameStatusCh <- GameStatusWaitHandStart
	}
}

//Pause 暂停
func (c *Holdem) Pause() {
	if !c.paused {
		c.log.Debug("pause")
		c.paused = true
		c.pauseCh = make(chan bool)
		c.seatLock.Lock()
		defer c.seatLock.Unlock()
		for _, rr := range c.roomers {
			rr.recv.RoomerGamePauseResume(c.id, true)
		}
	}
}

//Resume 继续
func (c *Holdem) Resume() {
	close(c.pauseCh)
	c.paused = false
}

//Cancel 提前取消
func (c *Holdem) Cancel() {
	if atomic.LoadInt32(&c.gameStartedLock) == int32(GameStatusNotStart) {
		c.gameStatusCh <- GameStatusCancel
		c.statusChange(GameStatusCancel)
	} else {
		c.log.Warn("can not cancel a started game")
	}
}

//ID 游戏标识
func (c *Holdem) ID() string {
	return c.id
}

//State 实时状态
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

//ForcePlayerStandUp 强制n个玩家起身
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
				c.standUp(seat, r, StandUpGameExchange)
			}
			num--
			if num == 0 {
				return
			}
		}
	}
}

//SendMessageTo 发送额外消息给游戏内某人
func (c *Holdem) SendMessageTo(code int, v interface{}, uids []string, r ...*Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	var uid string
	var seat int8
	if len(r) > 0 {
		uid = r[0].id
		if r[0].gameInfo != nil && r[0].gameInfo.seatNumber > 0 {
			seat = r[0].gameInfo.seatNumber
		}
	}
	sendMap := make(map[string]bool)
	for _, u := range uids {
		sendMap[u] = true
	}
	for _, rr := range c.roomers {
		//自己跳过
		if _, ok := sendMap[rr.id]; ok {
			if seat > 0 {
				rr.recv.RoomerMessage(c.id, code, v, uid, seat)
			} else {
				rr.recv.RoomerMessage(c.id, code, v, uid)
			}
		}
	}
}

//BroadcastMessage 广播消息通知
func (c *Holdem) BroadcastMessage(code int, v interface{}, r ...*Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	var uid string
	var seat int8
	if len(r) > 0 {
		uid = r[0].id
		if r[0].gameInfo != nil && r[0].gameInfo.seatNumber > 0 {
			seat = r[0].gameInfo.seatNumber
		}
	}
	for _, rr := range c.roomers {
		//自己跳过
		if rr.id == uid {
			continue
		}
		if seat > 0 {
			rr.recv.RoomerMessage(c.id, code, v, uid, seat)
		} else {
			rr.recv.RoomerMessage(c.id, code, v, uid)
		}
	}
}
