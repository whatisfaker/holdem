package holdem

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type Agent struct {
	user       UserInfo
	recv       Reciever
	gameInfo   *GameInfo
	betCh      chan *Bet
	atomAction int32
	//当前轮下注数量
	currentMaxBet int
	//最小加注数量
	minRaiseBet int
	//轮次(preflop, flop,turn,river)
	round     int8
	nextAgent *Agent
}

type ActionDef int8

const (
	ActionDefNone ActionDef = iota
	ActionDefSB
	ActionDefBB
	ActionDefBet
	ActionDefCall
	ActionDefFold
	ActionDefCheck
	ActionDefRaise
	ActionDefAllIn
)

func (c ActionDef) String() string {
	switch c {
	case ActionDefSB:
		return "small blind"
	case ActionDefBB:
		return "big blind"
	case ActionDefBet:
		return "bet"
	case ActionDefCall:
		return "call"
	case ActionDefFold:
		return "fold"
	case ActionDefCheck:
		return "check"
	case ActionDefRaise:
		return "raise"
	case ActionDefAllIn:
		return "all in"
	default:
		return "unknown"
	}
}

type Reciever interface {
	ID() string
	ErrorOccur(err error)
	//RoomerSeated 接收有人坐下
	RoomerSeated(int8, UserInfo)
	//RoomerRoomerStandUp
	RoomerStandUp(int8, UserInfo)
	//RoomerGetCard 接收有人收到牌（位置,牌数量)
	RoomerGetCard([]int8, int8)
	//RoomerGetPublicCard 接收公共牌
	RoomerGetPublicCard([]*Card)
	//RoomerGetAction 接收有人动作（位置，动作，金额(如果下注))
	RoomerGetAction(int8, ActionDef, ...int)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards([]*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult([]*Result)
	//PlayerGetCard 玩家获得自己发到的牌
	PlayerGetCard([]*Card)
	//PlayerCanBet 玩家可以开始下注
	PlayerCanBet()
}

func NewAgent(recv Reciever, user UserInfo) *Agent {
	agent := &Agent{
		user: user,
		recv: recv,
	}
	return agent
}

type Bet struct {
	Action ActionDef
	//Num 这次投入的数量
	Num int
	//RoundBet 这轮投入的数量
	RoundBet int
}

func (c *Agent) String() string {
	return fmt.Sprintf("chip:%d, roundBet:%d, handBet:%d", c.gameInfo.chip, c.gameInfo.roundBet, c.gameInfo.roundBet)
}

func (c *Agent) BringIn(chip int) {
	if chip <= 0 {
		c.recv.ErrorOccur(errors.New("chip is less than 0"))
	}
	c.gameInfo = &GameInfo{
		chip: chip,
	}
}

func (c *Agent) canAction() bool {
	return atomic.LoadInt32(&c.atomAction) == 1
}

func (c *Agent) enableAction(enable bool) {
	if enable {
		if atomic.LoadInt32(&c.atomAction) == 0 {
			atomic.AddInt32(&c.atomAction, 1)
		}
		return
	}
	if atomic.LoadInt32(&c.atomAction) == 1 {
		atomic.AddInt32(&c.atomAction, -1)
	}
}

func (c *Agent) Bet(bet *Bet) {
	if c.canAction() {
		c.gameInfo.status = bet.Action
		c.betCh <- bet
		c.enableAction(false)
	}
	c.recv.ErrorOccur(errors.New("it is not in bet time"))
}

type potSort []*Agent

func (p potSort) Len() int { return len(p) }

// 根据元素的年龄降序排序 （此处按照自己的业务逻辑写）
func (p potSort) Less(i, j int) bool {
	return p[i].gameInfo.handBet < p[j].gameInfo.handBet
}

// 交换数据
func (p potSort) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (c *Agent) waitBet(curBet int, minBet int, round int8, timeout time.Duration) *Bet {
	c.currentMaxBet = curBet
	c.minRaiseBet = minBet
	c.round = round
	c.enableAction(true)
	c.betCh = make(chan *Bet, 1)
	timer := time.NewTimer(timeout)
	defer func() {
		c.enableAction(false)
		close(c.betCh)
		timer.Stop()
	}()
	//稍微延迟告诉客户端可以下注
	time.AfterFunc(200*time.Millisecond, c.recv.PlayerCanBet)
	select {
	case bet, ok := <-c.betCh:
		if !ok {
			return nil
		}
		return bet
	case <-timer.C:
		c.gameInfo.status = ActionDefFold
		return &Bet{
			Action: ActionDefFold,
		}
	}
}

type ShowCard struct {
	SeatNumber int8
	Cards      []*Card
}
