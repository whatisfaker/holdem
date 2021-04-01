package holdem

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type ShowUser struct {
	User       UserInfo
	SeatNumber int8
	RoundBet   int
	Status     ActionDef
	IsPlaying  bool
}

type Agent struct {
	user       UserInfo
	log        *zap.Logger
	recv       Reciever
	gameInfo   *GameInfo
	betCh      chan *Bet
	atomAction int32
	showUser   *ShowUser
	nextAgent  *Agent
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
	ErrorOccur(int, error)
	//RoomerGameInformation 游戏信息
	RoomerGameInformation(*Holdem)
	//RoomerSeated 接收有人坐下
	RoomerSeated(int8, UserInfo)
	//RoomerRoomerStandUp
	RoomerStandUp(int8, UserInfo)
	//RoomerGetCard 接收有人收到牌（位置,牌数量)
	RoomerGetCard([]int8, int8)
	//RoomerGetPublicCard 接收公共牌
	RoomerGetPublicCard([]*Card)
	//RoomerGetAction 接收有人动作（按钮位, 位置，动作，金额(如果下注))
	RoomerGetAction(int8, int8, ActionDef, int)
	//RoomerGetShowCards 接收亮牌信息
	RoomerGetShowCards([]*ShowCard)
	//RoomerGetResult 接收牌局结果
	RoomerGetResult([]*Result)
	//PlayerGetCard 玩家获得自己发到的牌
	PlayerGetCard(int8, []*Card, []int8, int8)
	//PlayerCanBet 玩家可以开始下注(剩下筹码,本手已下注,本轮下注数量, 本轮的筹码数量, 最小下注额度)
	PlayerCanBet(seat int8, chip int, handBet int, roundBet int, curBet int, minBet int, round Round)
	//PlayerBringInSuccess 玩家带入成功
	PlayerBringInSuccess(seat int8, chip int)
	//PlayerSeated 玩家坐下
	PlayerSeated(int8)
	//PlayerStandUp 玩家站起
	PlayerStandUp(int8)
}

func NewAgent(recv Reciever, user UserInfo, log *zap.Logger) *Agent {
	agent := &Agent{
		user: user,
		recv: recv,
		log:  log,
	}
	return agent
}

type Bet struct {
	Action ActionDef
	//Num 这次投入的数量
	Num int
}

func (c *Agent) ErrorOccur(a int, e error) {
	c.recv.ErrorOccur(a, e)
}

func (c *Agent) String() string {
	return fmt.Sprintf("chip:%d, roundBet:%d, handBet:%d", c.gameInfo.chip, c.gameInfo.roundBet, c.gameInfo.roundBet)
}

func (c *Agent) ShowUser() *ShowUser {
	if c.gameInfo == nil {
		return nil
	}
	if c.showUser == nil {
		c.showUser = &ShowUser{
			User: c.user,
		}
	}
	if c.gameInfo == nil {
		return nil
	}
	c.showUser.SeatNumber = c.gameInfo.seatNumber
	if c.nextAgent != nil {
		c.showUser.RoundBet = c.gameInfo.roundBet
		c.showUser.Status = c.gameInfo.status
		c.showUser.IsPlaying = true
	} else {
		c.showUser.IsPlaying = false
	}
	return c.showUser
}

func (c *Agent) BringIn(chip int) {
	if chip <= 0 {
		c.ErrorOccur(ErrCodeLessChip, errLessChip)
		return
	}
	c.gameInfo = &GameInfo{
		chip: chip,
	}
	c.recv.PlayerBringInSuccess(0, chip)
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
		c.betCh <- bet
		return
	}
	c.ErrorOccur(ErrCodeNotInBetTime, errNotInBetTime)
}

type potSort []*Agent

func (p potSort) Len() int { return len(p) }

// 根据元素的年龄降序排序 （此处按照自己的业务逻辑写）
func (p potSort) Less(i, j int) bool {
	return p[i].gameInfo.handBet < p[j].gameInfo.handBet
}

// 交换数据
func (p potSort) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (c *Agent) waitBet(curBet int, minBet int, round Round, timeout time.Duration) *Bet {
	c.enableAction(true)
	c.betCh = make(chan *Bet, 1)
	timer := time.NewTimer(timeout)
	defer func() {
		c.enableAction(false)
		close(c.betCh)
		timer.Stop()
		c.log.Debug("bet end", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.String("round", round.String()))
	}()
	//稍微延迟告诉客户端可以下注
	time.AfterFunc(200*time.Millisecond, func() {
		c.log.Debug("wait bet", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.String("round", round.String()))
		c.recv.PlayerCanBet(c.gameInfo.seatNumber, c.gameInfo.chip, c.gameInfo.handBet, c.gameInfo.roundBet, curBet, minBet, round)
	})
	//循环如果投注错误,还可以让客户重新投注直到超时
	for {
		select {
		case bet, ok := <-c.betCh:
			if !ok {
				return nil
			}
			if c.isValidBet(bet, curBet, minBet, round) {
				c.gameInfo.status = bet.Action
				c.gameInfo.handBet += bet.Num
				c.gameInfo.roundBet += bet.Num
				c.gameInfo.chip -= bet.Num
				c.enableAction(false)
				return bet
			}
		case <-timer.C:
			c.gameInfo.status = ActionDefFold
			return &Bet{
				Action: ActionDefFold,
			}
		}
	}
}

func (c *Agent) isValidBet(bet *Bet, maxRoundBet int, minRaise int, round Round) bool {
	//第一个人/或者前面没有人下注
	actions := make(map[ActionDef]int)
	//c.log.Error("get bet", zap.Any("bet", bet))
	if maxRoundBet == 0 {
		if c.gameInfo.chip > minRaise {
			actions[ActionDefFold] = 0
			actions[ActionDefBet] = minRaise
			actions[ActionDefCheck] = 0
			actions[ActionDefAllIn] = c.gameInfo.chip
		} else {
			actions[ActionDefFold] = 0
			actions[ActionDefCheck] = 0
			actions[ActionDefAllIn] = c.gameInfo.chip
		}
	} else {
		//筹码大于当前下注
		if c.gameInfo.chip > maxRoundBet-c.gameInfo.roundBet+minRaise {
			actions[ActionDefFold] = 0
			actions[ActionDefCall] = maxRoundBet - c.gameInfo.roundBet
			actions[ActionDefRaise] = maxRoundBet - c.gameInfo.roundBet + minRaise
			actions[ActionDefAllIn] = c.gameInfo.chip
		} else if c.gameInfo.chip > maxRoundBet-c.gameInfo.roundBet {
			actions[ActionDefFold] = 0
			actions[ActionDefCall] = maxRoundBet - c.gameInfo.roundBet
			actions[ActionDefAllIn] = c.gameInfo.chip
		} else {
			actions[ActionDefFold] = 0
			actions[ActionDefAllIn] = c.gameInfo.chip
		}
	}
	amount, ok := actions[bet.Action]
	if !ok {
		c.recv.ErrorOccur(ErrCodeInvalidBetAction, errInvalidBetAction)
		return false
	}
	if (bet.Action == ActionDefRaise && bet.Num < amount) ||
		(bet.Action == ActionDefBet && bet.Num < amount) ||
		(bet.Action != ActionDefRaise && bet.Action != ActionDefBet && bet.Num != amount) {
		fmt.Println(bet.Action.String(), bet.Num, amount, maxRoundBet, c.gameInfo.roundBet, minRaise)
		c.recv.ErrorOccur(ErrCodeInvalidBetNum, errInvalidBetNum)
		return false
	}
	return true
}

type ShowCard struct {
	SeatNumber int8
	Cards      []*Card
}
