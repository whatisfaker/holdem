package holdem

import (
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type ShowUser struct {
	ID         string
	SeatNumber int8
	Chip       uint
	RoundBet   uint
	Status     ActionDef
	HandNum    uint
	Te         PlayType
	Cards      []*Card //坐着的用户返回信息带卡牌信息
}

type Agent struct {
	id                string
	auto              bool
	log               *zap.Logger
	recv              Reciever
	h                 *Holdem
	gameInfo          *gameInfo
	betCh             chan *Bet
	insuranceCh       chan []*BuyInsurance
	atomBetLock       int32
	atomInsuranceLock int32
	showUser          *ShowUser
	nextAgent         *Agent
	prevAgent         *Agent
	fake              bool
}

func NewAgent(recv Reciever, id string, log *zap.Logger) *Agent {
	agent := &Agent{
		id:   id,
		recv: recv,
		log:  log,
	}
	return agent
}

func (c *Agent) replace(rs *Agent) {
	//托管状态覆盖
	c.auto = rs.auto
	c.recv = rs.recv
}

type Bet struct {
	Action ActionDef
	//Num 这次投入的数量
	Num uint
	//Auto
	Auto bool
}

type BuyInsurance struct {
	Card *Card
	Num  uint
}

type InsuranceResult struct {
	//SeatNumber 座位号
	SeatNumber int8
	//Cost 消费
	Cost uint
	//Earn 获取
	Earn float64
	//Outs 补牌数
	Outs int
	//Round 回合
	Round Round
}

func (c *Agent) ID() string {
	return c.id
}

func (c *Agent) EnableAuto() {
	c.auto = true
}

func (c *Agent) DisableAuto() {
	c.auto = false
}

func (c *Agent) ErrorOccur(a int, e error) {
	c.log.Error("error", zap.Error(e), zap.String("id", c.id))
	c.recv.ErrorOccur(a, e)
}

//ShowUser 展示用户信息
func (c *Agent) displayUser(showCards bool) *ShowUser {
	if c.gameInfo == nil {
		return nil
	}
	if c.showUser == nil {
		c.showUser = &ShowUser{
			ID: c.id,
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
	if showCards {
		c.showUser.Cards = c.gameInfo.cards
	}
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
// func (c *Agent) Info() {
// 	if c.h == nil {
// 		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
// 		return
// 	}
// 	s := c.h.State()
// 	for k := range s.Seated {
// 		p := s.Seated[k]
// 		if p.ID == c.id {
// 			p.Cards = c.gameInfo.cards
// 		}
// 		s.Seated[k] = p
// 	}
// 	c.recv.RoomerGameInformation(c, s)
// }

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
func (c *Agent) BringIn(chip uint) {
	if c.h == nil {
		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
		return
	}
	if c.h.status() == GameStatusComplete || c.h.status() == GameStatusCancel {
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
		c.gameInfo = &gameInfo{
			chip:    chip,
			bringIn: chip,
		}
	}
	c.log.Debug("user bring in", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("id", c.id), zap.Uint("bringin", chip))
	c.recv.PlayerBringInSuccess(c.gameInfo.seatNumber, chip)
}

//Seated 坐下（不输入座位号,自动寻座)
func (c *Agent) Seated(i ...int8) {
	if c.h == nil {
		c.ErrorOccur(ErrCodeNoJoin, errNoJoin)
		return
	}
	if c.h.status() == GameStatusComplete || c.h.status() == GameStatusCancel {
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
	if len(i) > 0 {
		c.h.seated(i[0], c)
		return
	}
	//auto find seat
	c.h.seated(0, c)
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
	c.gameInfo.needStandUpReason = StandUpAction
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
	var amount uint
	defer func() {
		c.enableBuyInsurance(false)
		close(c.insuranceCh)
		timer.Stop()
		c.log.Debug("buy insurance end", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.Uint("amount", amount), zap.String("round", round.String()))
	}()
	//稍微延迟告诉客户端可以下注
	time.AfterFunc(delaySend, func() {
		c.log.Debug("wait buy insurance", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.String("round", round.String()), zap.Int("outslen", outsLen))
		c.recv.PlayerCanBuyInsurance(c.gameInfo.seatNumber, outsLen, odds, outs, round)
	})
	if c.auto {
		return nil, nil
	}
	//循环如果投注错误,还可以让客户重新投注直到超时
	for {
		select {
		case is, ok := <-c.insuranceCh:
			if !ok {
				return nil, nil
			}
			var cost uint
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
			c.gameInfo.autoTimes++
			if c.gameInfo.autoTimes >= 4 {
				c.auto = true
			}
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

// func (p betSort) GroupBet() []map[int8]bool {
// 	sort.Sort(p)
// 	var pot map[int8]bool
// 	pots := make([]map[int8]bool, 0)
// 	var num uint
// 	for _, a := range p {
// 		if pot == nil {
// 			pot = map[int8]bool{a.gameInfo.seatNumber: true}
// 			num = a.gameInfo.handBet
// 			continue
// 		}
// 		if a.gameInfo.handBet == num {
// 			pot[a.gameInfo.seatNumber] = true
// 		} else {
// 			pots = append(pots, pot)
// 			pot = map[int8]bool{a.gameInfo.seatNumber: true}
// 			num = a.gameInfo.handBet
// 		}
// 	}
// 	pots = append(pots, pot)
// 	return pots
// }

func (c *Agent) waitBet(curBet uint, minRaise uint, round Round, timeout time.Duration) (rbet *Bet) {
	c.enableBet(true)
	c.betCh = make(chan *Bet, 1)
	timer := time.NewTimer(timeout)
	defer func() {
		c.enableBet(false)
		close(c.betCh)
		timer.Stop()
		c.log.Debug("bet end", zap.Int8("seat", c.gameInfo.seatNumber), zap.String("status", c.gameInfo.status.String()), zap.Uint("amount", rbet.Num), zap.Bool("auto", rbet.Auto), zap.String("round", round.String()))
	}()
	//托管直接操作
	if c.auto {
		c.gameInfo.status = ActionDefCheck
		rbet = &Bet{
			Action: ActionDefCheck,
			Auto:   true,
		}
		//无法check就fold
		if valid, _ := c.isValidBet(rbet, curBet, minRaise, round); !valid {
			rbet.Action = ActionDefFold
			c.gameInfo.status = ActionDefFold
		}
		return
	}
	//循环如果投注错误,还可以让客户重新投注直到超时
	for {
		select {
		case bet, ok := <-c.betCh:
			if !ok {
				return nil
			}
			if valid, err2 := c.isValidBet(bet, curBet, minRaise, round); valid {
				c.gameInfo.status = bet.Action
				c.gameInfo.handBet += bet.Num
				c.gameInfo.roundBet += bet.Num
				c.gameInfo.chip -= bet.Num
				c.enableBet(false)
				rbet = bet
				return
			} else {
				c.log.Error("invalid bet num", zap.String("action", bet.Action.String()), zap.Uint("num", bet.Num), zap.Uint("maxbet", curBet), zap.Uint("mybeted", c.gameInfo.roundBet), zap.Uint("min_raise", minRaise), zap.Uint("mychip", c.gameInfo.chip))
				c.ErrorOccur(err2.code, err2.err)
			}
		case <-timer.C:
			//超时尝试check
			c.gameInfo.status = ActionDefCheck
			rbet = &Bet{
				Action: ActionDefCheck,
				Auto:   true,
			}
			//无法check就fold
			if valid, _ := c.isValidBet(rbet, curBet, minRaise, round); !valid {
				rbet.Action = ActionDefFold
				c.gameInfo.status = ActionDefFold
			}
			c.gameInfo.autoTimes++
			if c.gameInfo.autoTimes >= 4 {
				c.auto = true
			}
			return
		}
	}
}

//isValidBet 判断是否是有效的投注
func (c *Agent) isValidBet(bet *Bet, maxRoundBet uint, minRaise uint, round Round) (bool, *errorWithCode) {
	//第一个人/或者前面没有人下注
	actions := make(map[ActionDef]uint)
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
		return false, &errorWithCode{
			code: ErrCodeInvalidBetAction,
			err:  errInvalidBetAction,
		}
	}
	if (bet.Action == ActionDefRaise && bet.Num < amount) ||
		(bet.Action == ActionDefBet && bet.Num < amount) ||
		(bet.Action != ActionDefRaise && bet.Action != ActionDefBet && bet.Num != amount) {
		return false, &errorWithCode{
			code: ErrCodeInvalidBetNum,
			err:  errInvalidBetNum,
		}
	}
	return true, nil
}
