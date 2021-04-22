package holdem

import (
	"sort"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type ShowUser struct {
	User       UserInfo
	SeatNumber int8
	Chip       int
	RoundBet   int
	Status     ActionDef
	HandNum    int
	Te         PlayType
	Cards      []*Card //坐着的用户返回信息带卡牌信息
}

type Agent struct {
	//	invalid           bool
	user              UserInfo
	log               *zap.Logger
	recv              Reciever
	h                 *Holdem
	gameInfo          *GameInfo
	betCh             chan *Bet
	insuranceCh       chan []*BuyInsurance
	atomBetLock       int32
	atomInsuranceLock int32
	showUser          *ShowUser
	nextAgent         *Agent
	prevAgent         *Agent
	fake              bool
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
	//Auto
	Auto bool
}

type BuyInsurance struct {
	Card *Card
	Num  int
}

type InsuranceResult struct {
	//SeatNumber 座位号
	SeatNumber int8
	//Cost 消费
	Cost int
	//Earn 获取
	Earn float64
	//Outs 补牌数
	Outs int
	//Round 回合
	Round Round
}

func (c *Agent) ErrorOccur(a int, e error) {
	c.log.Error("error", zap.Error(e), zap.String("id", c.user.ID()))
	c.recv.ErrorOccur(a, e)
}

// func (c *Agent) String() string {
// 	return fmt.Sprintf("chip:%d, roundBet:%d, handBet:%d", c.gameInfo.chip, c.gameInfo.roundBet, c.gameInfo.roundBet)
// }

//ShowUser 展示用户信息
func (c *Agent) displayUser() *ShowUser {
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
	c.showUser.Chip = c.gameInfo.chip
	c.showUser.SeatNumber = c.gameInfo.seatNumber
	c.showUser.RoundBet = c.gameInfo.roundBet
	c.showUser.HandNum = c.gameInfo.handNum
	c.showUser.Status = c.gameInfo.status
	c.showUser.Te = c.gameInfo.te
	return c.showUser
}

//Join 加入游戏
func (c *Agent) Join(holdem *Holdem) {
	if c.h == holdem {
		return
	}
	holdem.join(c)
	c.h = holdem
	c.gameInfo = nil
}

//Info 获取信息
func (c *Agent) Info() {
	if c.h == nil {
		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
		return
	}
	s := c.h.state()
	for k := range s.Seated {
		p := s.Seated[k]
		if p.User.ID() == c.user.ID() && p.User.Name() == c.user.Name() {
			p.Cards = c.gameInfo.cards
		}
		s.Seated[k] = p
	}
	c.recv.RoomerGameInformation(s)
}

//Leave 离开
func (c *Agent) Leave(holdem *Holdem) {
	if c.h == nil {
		return
	}
	if c.gameInfo != nil {
		c.ErrorOccur(ErrCodeNotStandUp, errNotStandUp)
		return
	}
	holdem.leave(c)
	c.h = nil
}

// func (c *Agent) Invalid(b bool) {
// 	c.invalid = b
// }

//BringIn 带入筹码
func (c *Agent) BringIn(chip int) {
	if c.h == nil {
		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
		return
	}
	if c.h.gameStatus == GameStatusComplete {
		c.ErrorOccur(ErrCodeGameOver, errGameOver)
		return
	}
	if chip <= 0 {
		c.ErrorOccur(ErrCodeLessChip, errLessChip)
		return
	}
	if c.gameInfo != nil {
		c.gameInfo.bringIn += chip
		c.gameInfo.chip += chip
	} else {
		c.gameInfo = &GameInfo{
			chip:    chip,
			bringIn: chip,
		}
	}
	c.log.Debug("user bring in", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("id", c.user.ID()), zap.Int("bringin", chip))
	c.recv.PlayerBringInSuccess(c.gameInfo.seatNumber, chip)
}

//Seated 坐下
func (c *Agent) Seated(i int8) {
	if c.h == nil {
		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
		return
	}
	if c.h.gameStatus == GameStatusComplete {
		c.ErrorOccur(ErrCodeGameOver, errGameOver)
		return
	}
	if c.gameInfo == nil {
		c.ErrorOccur(ErrCodeNotPlaying, errNotPlaying)
		return
	}
	if c.gameInfo.seatNumber > 0 {
		c.ErrorOccur(ErrCodeAlreadySeated, errAlreadySeated)
		return
	}
	c.h.seated(i, c)
}

//StandUp 站起来
func (c *Agent) StandUp() {
	if c.gameInfo == nil {
		c.ErrorOccur(ErrCodeNotPlaying, errNotPlaying)
		return
	}
	if c.gameInfo.seatNumber <= 0 {
		c.ErrorOccur(ErrCodeNoSeat, errNoSeat)
		return
	}
	c.gameInfo.needStandUp = true
	if c.gameInfo.status == ActionDefNone {
		c.h.directStandUp(c.gameInfo.seatNumber, c)
		return
	}
	c.recv.PlayerReadyStandUpSuccess(c.gameInfo.seatNumber)
}

//Bet 下注
func (c *Agent) Bet(bet *Bet) {
	if c.canBet() {
		c.betCh <- bet
		return
	}
	c.ErrorOccur(ErrCodeNotInBetTime, errNotInBetTime)
}

//PayToPlay 补盲
func (c *Agent) PayToPlay() {
	if c.gameInfo == nil {
		c.ErrorOccur(ErrCodeNotPlaying, errNotPlaying)
		return
	}
	if c.gameInfo.seatNumber <= 0 {
		c.ErrorOccur(ErrCodeNoSeat, errNoSeat)
		return
	}
	if c.gameInfo.te == PlayTypeDisable {
		c.ErrorOccur(ErrCodeCannotEnablePayToPlay, errCannotEnablePayToPlay)
		return
	}
	if c.gameInfo.te == PlayTypeNeedPayToPlay {
		c.gameInfo.te = PlayTypeAgreePayToPlay
	}
	c.recv.PlayerPayToPlaySuccesss(c.gameInfo.seatNumber)
}

func (c *Agent) canBet() bool {
	return atomic.LoadInt32(&c.atomBetLock) == 1
}

func (c *Agent) enableBet(enable bool) {
	if enable {
		if atomic.LoadInt32(&c.atomBetLock) == 0 {
			atomic.AddInt32(&c.atomBetLock, 1)
		}
		return
	}
	if atomic.LoadInt32(&c.atomBetLock) == 1 {
		atomic.AddInt32(&c.atomBetLock, -1)
	}
}

//Bet 下注
func (c *Agent) BuyInsurance(insurance []*BuyInsurance) {
	if c.canBuyInsurance() {
		c.insuranceCh <- insurance
		return
	}
	c.ErrorOccur(ErrCodeNotInBetTime, errNotInBetTime)
}

func (c *Agent) canBuyInsurance() bool {
	return atomic.LoadInt32(&c.atomInsuranceLock) == 1
}

func (c *Agent) enableBuyInsurance(enable bool) {
	if enable {
		if atomic.LoadInt32(&c.atomInsuranceLock) == 0 {
			atomic.AddInt32(&c.atomInsuranceLock, 1)
		}
		return
	}
	if atomic.LoadInt32(&c.atomInsuranceLock) == 1 {
		atomic.AddInt32(&c.atomInsuranceLock, -1)
	}
}

func (c *Agent) waitBuyInsurance(outsLen int, odds float64, outs map[int8][]*UserOut, round Round, timeout time.Duration) (*InsuranceResult, []*BuyInsurance) {
	c.enableBuyInsurance(true)
	c.insuranceCh = make(chan []*BuyInsurance, 1)
	c.gameInfo.insurance = make(map[int8]*BuyInsurance)
	timer := time.NewTimer(timeout)
	amount := 0
	defer func() {
		c.enableBuyInsurance(false)
		close(c.insuranceCh)
		timer.Stop()
		c.log.Debug("buy insurance end", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.Int("amount", amount), zap.String("round", round.String()))
	}()
	//稍微延迟告诉客户端可以下注
	time.AfterFunc(200*time.Millisecond, func() {
		c.log.Debug("wait buy insurance", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.String("round", round.String()), zap.Int("outslen", outsLen))
		c.recv.PlayerCanBuyInsurance(c.gameInfo.seatNumber, outsLen, odds, outs, round)
	})
	//循环如果投注错误,还可以让客户重新投注直到超时
	for {
		select {
		case is, ok := <-c.insuranceCh:
			if !ok {
				return nil, nil
			}
			cost := 0
			for _, v := range is {
				c.gameInfo.insurance[v.Card.Value()] = v
				cost += v.Num
			}
			if cost < c.gameInfo.chip {
				c.ErrorOccur(ErrCodeInvalidInsurance, errInvalidInsurance)
				continue
			}
			amount = cost
			return &InsuranceResult{
				SeatNumber: c.gameInfo.seatNumber,
				Round:      round,
				Cost:       cost,
				Outs:       outsLen,
			}, is
		case <-timer.C:
			return nil, nil
		}
	}
}

type betSort []*Agent

func (p betSort) Len() int { return len(p) }

func (p betSort) Less(i, j int) bool {
	return p[i].gameInfo.handBet < p[j].gameInfo.handBet
}

func (p betSort) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

//BetGroup 下注分组
type BetGroup struct {
	//SeatNumber 相同下注的座位号
	SeatNumber map[int8]bool
	//Num 数量
	Num int
}

func (p betSort) GroupBet() []map[int8]bool {
	sort.Sort(p)
	var pot map[int8]bool
	pots := make([]map[int8]bool, 0)
	var num int
	for _, a := range p {
		if pot == nil {
			pot = map[int8]bool{a.gameInfo.seatNumber: true}
			num = a.gameInfo.handBet
			continue
		}
		if a.gameInfo.handBet == num {
			pot[a.gameInfo.seatNumber] = true
		} else {
			pots = append(pots, pot)
			pot = map[int8]bool{a.gameInfo.seatNumber: true}
			num = a.gameInfo.handBet
		}
	}
	pots = append(pots, pot)
	return pots
}

func (c *Agent) waitBet(curBet int, minBet int, round Round, timeout time.Duration) (rbet *Bet) {
	c.enableBet(true)
	c.betCh = make(chan *Bet, 1)
	timer := time.NewTimer(timeout)
	defer func() {
		c.enableBet(false)
		close(c.betCh)
		timer.Stop()
		c.log.Debug("bet end", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.Int("amount", rbet.Num), zap.Bool("auto", rbet.Auto), zap.String("round", round.String()))
	}()
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
				c.enableBet(false)
				rbet = bet
				return
			}
		case <-timer.C:
			c.gameInfo.status = ActionDefFold
			rbet = &Bet{
				Action: ActionDefFold,
				Auto:   true,
			}
			return
		}
	}
}

//isValidBet 判断是否是有效的投注
func (c *Agent) isValidBet(bet *Bet, maxRoundBet int, minRaise int, round Round) bool {
	//第一个人/或者前面没有人下注
	actions := make(map[ActionDef]int)
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
		c.log.Error("invalid bet action", zap.String("action", bet.Action.String()), zap.Int("num", bet.Num), zap.Int("maxbet", maxRoundBet), zap.Int("mybeted", c.gameInfo.roundBet), zap.Int("min_raise", minRaise), zap.Int("mychip", c.gameInfo.chip))
		c.ErrorOccur(ErrCodeInvalidBetAction, errInvalidBetAction)
		return false
	}
	if (bet.Action == ActionDefRaise && bet.Num < amount) ||
		(bet.Action == ActionDefBet && bet.Num < amount) ||
		(bet.Action != ActionDefRaise && bet.Action != ActionDefBet && bet.Num != amount) {
		c.log.Error("invalid bet num", zap.String("action", bet.Action.String()), zap.Int("num", bet.Num), zap.Int("maxbet", maxRoundBet), zap.Int("mybeted", c.gameInfo.roundBet), zap.Int("min_raise", minRaise), zap.Int("mychip", c.gameInfo.chip))
		c.ErrorOccur(ErrCodeInvalidBetNum, errInvalidBetNum)
		return false
	}
	return true
}
