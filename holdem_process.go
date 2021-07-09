package holdem

import (
	"time"

	"go.uber.org/zap"
)

//doAnte 前注
func (c *Holdem) doAnte() {
	u := c.button
	for {
		if u.gameInfo.te == PlayTypeDisable || u.gameInfo.te == PlayTypeNeedPayToPlay {
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		if u.gameInfo.chip >= c.ante {
			c.pot += c.ante
			u.gameInfo.chip -= c.ante
			u.gameInfo.status = ActionDefAnte
			c.options.recorder.Ante(c.base(), u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, c.ante)
			c.log.Debug("ante", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", c.ante))
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		c.handStartInfo.AnteAllIns = append(c.handStartInfo.AnteAllIns, u.gameInfo.seatNumber)
		c.pot += u.gameInfo.chip
		c.options.recorder.Ante(c.base(), u.gameInfo.seatNumber, u.ID(), 0, u.gameInfo.chip)
		u.gameInfo.chip = 0
		u.gameInfo.status = ActionDefAllIn
		c.log.Debug("ante", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", u.gameInfo.chip))
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
}

//smallBlind 小盲
func (c *Holdem) smallBlind() {
	u := c.button.nextAgent
	if u.gameInfo.te == PlayTypeDisable || u.gameInfo.te == PlayTypeNeedPayToPlay {
		c.log.Debug("small blind(empty)", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int8("play type", int8(u.gameInfo.te)))
		return
	}
	if u.gameInfo.status == ActionDefAllIn {
		c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("allin", 0))
		return
	}
	if u.gameInfo.chip >= c.sb {
		c.pot += c.sb
		u.gameInfo.roundBet = c.sb
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefSB
		c.handStartInfo.SB = &Bet{
			Action: ActionDefSB,
			Num:    c.sb,
		}
		c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, ActionDefSB, c.sb)
		c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", c.sb))
		return
	}
	//不够时 全下
	c.pot += u.gameInfo.chip
	u.gameInfo.roundBet = u.gameInfo.chip
	u.gameInfo.handBet += u.gameInfo.roundBet
	u.gameInfo.chip -= u.gameInfo.roundBet
	u.gameInfo.status = ActionDefAllIn
	c.handStartInfo.SB = &Bet{
		Action: ActionDefAllIn,
		Num:    u.gameInfo.roundBet,
	}
	c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, ActionDefAllIn, u.gameInfo.roundBet)
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", u.gameInfo.roundBet))
}

//bigBlind 大盲
func (c *Holdem) bigBlind() {
	u := c.button.nextAgent.nextAgent
	if u.gameInfo.status == ActionDefAllIn {
		c.log.Debug("big blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("allin", 0))
		return
	}
	if u.gameInfo.chip >= 2*c.sb {
		c.pot += c.sb * 2
		u.gameInfo.roundBet = c.sb * 2
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefBB
		c.handStartInfo.BB = &Bet{
			Action: ActionDefBB,
			Num:    c.sb * 2,
		}
		c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, ActionDefBB, 2*c.sb)
		c.log.Debug("big blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", c.sb*2))
		return
	}
	//不够时 全下
	c.pot += u.gameInfo.chip
	u.gameInfo.roundBet = u.gameInfo.chip
	u.gameInfo.handBet += u.gameInfo.roundBet
	u.gameInfo.chip -= u.gameInfo.roundBet
	u.gameInfo.status = ActionDefAllIn
	c.handStartInfo.BB = &Bet{
		Action: ActionDefAllIn,
		Num:    u.gameInfo.roundBet,
	}
	c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, ActionDefAllIn, u.gameInfo.roundBet)
	c.log.Debug("big blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", u.gameInfo.roundBet))
}

func (c *Holdem) payToPlay() {
	u := c.button
	for {
		if u.gameInfo.te == PlayTypeAgreePayToPlay {
			c.pot += c.sb * 2
			u.gameInfo.roundBet = c.sb * 2
			u.gameInfo.handBet += u.gameInfo.roundBet
			u.gameInfo.chip -= u.gameInfo.roundBet
			u.gameInfo.status = ActionDefBB
			u.gameInfo.te = PlayTypeNormal
			c.handStartInfo.PayToPlay = append(c.handStartInfo.PayToPlay, u.gameInfo.seatNumber)
			//补盲
			c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, ActionDefBB, 2*c.sb)
			c.log.Debug("pay to play", zap.Int8("seat", u.gameInfo.seatNumber), zap.Uint("amount", c.sb*2))
		}
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
}

//deal 发牌
func (c *Holdem) deal() *Agent {
	cnt := 2
	c.log.Debug("deal begin", zap.Int("cards_count", 2))
	first := c.button.nextAgent
	cards := make([][]*Card, c.playingPlayerCount)
	max := cnt
	for ; max > 0; max-- {
		for i := 0; i < int(c.playingPlayerCount); i++ {
			cds, _ := c.poker.GetCards(1)
			if len(cards[i]) == 0 {
				cards[i] = make([]*Card, 0)
			}
			cards[i] = append(cards[i], cds...)
		}
	}
	cur := first
	firstAg := c.getNextOpAgent(c.button.nextAgent.nextAgent)
	op := newOperator(firstAg, 2*c.sb, 2*c.sb, c.waitBetTimeout)
	if firstAg != nil {
		firstAg.enableBet(true)
	}
	c.addWaitTime(c.waitBetTimeout)
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	i := 0
	seats := make([]int8, 0)
	for {
		//在座，但是不能发牌
		if cur.fake {
			cur = cur.nextAgent
		} else {
			cur.gameInfo.cards = cards[i]
			cur.recv.PlayerGetCard(c.id, cur.gameInfo.seatNumber, cur.id, cards[i], seats, int8(cnt), c.handStartInfo, op)
			i++
			seats = append(seats, cur.gameInfo.seatNumber)
			cur = cur.nextAgent
		}
		if cur == first {
			break
		}
	}
	for _, r := range c.roomers {
		if r.gameInfo == nil {
			r.recv.RoomerGetCard(c.id, seats, int8(cnt), c.handStartInfo, op)
		}
	}
	c.log.Debug("deal end")
	return firstAg
}

//preflop 翻牌前叫注
func (c *Holdem) preflop(op *Agent) ([]*Agent, bool) {
	c.statusChange(GameStatusHandPreflop)
	c.roundBet = c.sb * 2
	c.minRaise = c.sb * 2
	u := op
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	c.log.Debug(RoundPreFlop.String()+" bet begin", zap.Int8("pc", c.playingPlayerCount), zap.Int8("sseat", u.gameInfo.seatNumber), zap.String("suser", u.ID()))
	for u != nil {
		c.waitPause()
		c.log.Debug("wait bet", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", RoundPreFlop.String()))
		bet := u.waitBet(c.roundBet, c.minRaise, RoundPreFlop, c.waitBetTimeout+delaySend)
		switch bet.Action {
		case ActionDefFold:
			//盖牌的直接移除出局
			u2 := fakeAgent(u)
			if u == c.button {
				c.button = u2
			}
			//如果要离开直接让他离开
			if u.gameInfo.needStandUpReason != StandUpNone {
				c.seatLock.Lock()
				c.standUp(u.gameInfo.seatNumber, u, u.gameInfo.needStandUpReason)
				c.seatLock.Unlock()
			}
			u = u2
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = u.gameInfo.roundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = u.gameInfo.roundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := u.gameInfo.roundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if u.gameInfo.roundBet > c.roundBet {
				c.roundBet = u.gameInfo.roundBet
			}
		default:
			c.log.Error("incorrect action", zap.String("action", bet.Action.String()))
			panic("incorrect action")
		}
		c.options.recorder.Action(c.base(), RoundPreFlop, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, bet.Action, bet.Num)
		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		var next *Agent
		var op *Operator
		if !roundComplete {
			next = c.getNextOpAgent(u)
			op = newOperator(next, c.roundBet, c.minRaise, c.waitBetTimeout)
			if next != nil {
				next.enableBet(true)
			}
		}
		c.addWaitTime(c.waitBetTimeout)
		//稍微延迟告诉客户端可以下注
		u.recv.PlayerActionSuccess(c.id, u.gameInfo.seatNumber, u.id, bet.Action, bet.Num, op)
		c.seatLock.Lock()
		for uid, r := range c.roomers {
			if uid != u.id {
				r.recv.RoomerGetAction(c.id, u.gameInfo.seatNumber, u.id, bet.Action, bet.Num, op)
			}
		}
		c.seatLock.Unlock()
		u = next
	}
	//等500ms
	time.Sleep(2 * delaySend)
	if showcard {
		scs := make([]*ShowCard, 0)
		for _, v := range unfoldUsers {
			scs = append(scs, &ShowCard{
				SeatNumber: v.gameInfo.seatNumber,
				ID:         v.id,
				Cards:      v.gameInfo.cards,
			})
		}
		c.seatLock.Lock()
		for _, r := range c.roomers {
			r.recv.RoomerGetShowCards(c.id, scs)
		}
		c.seatLock.Unlock()
	}
	c.log.Debug(RoundPreFlop.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//dealPublicCards 发公共牌
func (c *Holdem) dealPublicCards(n int, round Round) ([]*Card, *Agent) {
	c.log.Debug("deal public cards(start)", zap.Int("cards_count", n))
	//洗牌
	_, _ = c.poker.GetCards(1)
	cards, _ := c.poker.GetCards(n)
	c.publicCards = append(c.publicCards, cards...)
	firstAg := c.getNextOpAgent(c.button)
	firstOp := newOperator(firstAg, 0, 2*c.sb, c.waitBetTimeout)
	c.addWaitTime(c.waitBetTimeout)
	if firstAg != nil {
		firstAg.enableBet(true)
	}
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	for _, r := range c.roomers {
		r.recv.RoomerGetPublicCard(c.id, cards, firstOp)
	}
	c.log.Debug("deal public cards(end)", zap.Int("cards_count", n))
	return cards, firstAg
}

//flopTurnRiver 叫注（轮描述）
func (c *Holdem) flopTurnRiver(u *Agent, round Round) ([]*Agent, bool) {
	switch round {
	case RoundFlop:
		c.statusChange(GameStatusHandFlop)
	case RoundTurn:
		c.statusChange(GameStatusHandTurn)
	case RoundRiver:
		c.statusChange(GameStatusHandRiver)
	}
	c.roundBet = 0
	c.minRaise = c.sb * 2
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	//清理此轮
	uu := c.button
	for {
		uu.gameInfo.roundBet = 0
		uu = uu.nextAgent
		if uu == c.button {
			break
		}
	}
	c.log.Debug(round.String()+" bet begin", zap.Int8("pc", c.playingPlayerCount), zap.Int8("sseat", u.gameInfo.seatNumber), zap.String("suser", u.ID()))
	for u != nil {
		c.waitPause()
		c.log.Debug("wait bet", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", round.String()))
		bet := u.waitBet(c.roundBet, c.minRaise, round, c.waitBetTimeout+delaySend)
		switch bet.Action {
		case ActionDefFold:
			//盖牌的直接移除出局
			u2 := fakeAgent(u)
			if u == c.button {
				c.button = u2
			}
			//如果要离开直接让他离开
			if u.gameInfo.needStandUpReason != StandUpNone {
				c.seatLock.Lock()
				c.standUp(u.gameInfo.seatNumber, u, u.gameInfo.needStandUpReason)
				c.seatLock.Unlock()
			}
			u = u2
		case ActionDefCheck:
		case ActionDefBet:
			c.pot += bet.Num
			c.roundBet = bet.Num
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = u.gameInfo.roundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = u.gameInfo.roundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := u.gameInfo.roundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if u.gameInfo.roundBet > c.roundBet {
				c.roundBet = u.gameInfo.roundBet
			}
		default:
			c.log.Error("incorrect action", zap.Int8("action", int8(bet.Action)))
			panic("incorrect action")
		}
		c.options.recorder.Action(c.base(), round, u.gameInfo.seatNumber, u.ID(), u.gameInfo.chip, bet.Action, bet.Num)
		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		var next *Agent
		var op *Operator
		if !roundComplete {
			next = c.getNextOpAgent(u)
			op = newOperator(next, c.roundBet, c.minRaise, c.waitBetTimeout)
			if next != nil {
				next.enableBet(true)
			}
		}
		//稍微延迟告诉客户端可以下注
		c.addWaitTime(c.waitBetTimeout)
		u.recv.PlayerActionSuccess(c.id, u.gameInfo.seatNumber, u.id, bet.Action, bet.Num, op)
		c.seatLock.Lock()
		for uid, r := range c.roomers {
			if uid != u.ID() {
				r.recv.RoomerGetAction(c.id, u.gameInfo.seatNumber, u.id, bet.Action, bet.Num, op)
			}
		}
		c.seatLock.Unlock()
		u = next
	}
	time.Sleep(2 * delaySend)
	//非河牌直接亮牌
	if round != RoundRiver && showcard {
		scs := make([]*ShowCard, 0)
		for _, v := range unfoldUsers {
			scs = append(scs, &ShowCard{
				SeatNumber: v.gameInfo.seatNumber,
				Cards:      v.gameInfo.cards,
			})
		}
		c.seatLock.Lock()
		for _, r := range c.roomers {
			r.recv.RoomerGetShowCards(c.id, scs)
		}
		c.seatLock.Unlock()
	}
	c.log.Debug(round.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//complexWin 斗牌结算
func (c *Holdem) complexWin(users []*Agent) {
	pots := c.calcPot(users)
	results, _, _ := c.calcWin(users, pots)
	c.pot = 0
	ret := make([]*Result, 0)
	u := c.button
	//所有玩家的最终状况
	for {
		r := &Result{
			SeatNumber: u.gameInfo.seatNumber,
		}
		if u.gameInfo.cardResults != nil {
			r.Cards = u.gameInfo.cardResults
			r.HandValueType = u.gameInfo.handValue.MaxHandValueType()
		}
		if rv, ok := results[u.gameInfo.seatNumber]; ok {
			u.gameInfo.chip += rv.Num
			r.Num = rv.Num
		}
		//保险
		if iv, ok := c.insuranceResult[u.gameInfo.seatNumber]; ok {
			r.InsuranceResult = iv
		}
		r.Chip = u.gameInfo.chip
		ret = append(ret, r)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	c.seatLock.Lock()
	for _, r := range c.roomers {
		r.recv.RoomerGetResult(c.id, ret)
	}
	c.statusChange(GameStatusHandEnd)
	c.options.recorder.HandEnd(c.information(), ret)
	c.seatLock.Unlock()
	c.log.Debug("cwin", zap.Any("result", ret))
}

//simpleWin 单人获胜（只有一人未盖牌)
func (c *Holdem) simpleWin(agent *Agent) {
	ret := make([]*Result, 0)
	u := c.button
	for {
		r := &Result{
			SeatNumber: u.gameInfo.seatNumber,
			Te:         u.gameInfo.te,
		}
		if u.gameInfo.seatNumber == agent.gameInfo.seatNumber {
			u.gameInfo.chip += c.pot
			r.Num = c.pot
			c.pot = 0
		}
		//保险
		if iv, ok := c.insuranceResult[u.gameInfo.seatNumber]; ok {
			r.InsuranceResult = iv
		}
		r.Chip = u.gameInfo.chip
		ret = append(ret, r)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	c.seatLock.Lock()
	for _, r := range c.roomers {
		r.recv.RoomerGetResult(c.id, ret)
	}
	c.statusChange(GameStatusHandEnd)
	c.options.recorder.HandEnd(c.information(), ret)
	c.seatLock.Unlock()
	c.log.Debug("swin", zap.Int8("seat", agent.gameInfo.seatNumber), zap.String("user", agent.ID()), zap.Any("result", ret))
}

//StartHand 开始新的一手
func (c *Holdem) startHand() {
	c.waitPause()
	c.statusChange(GameStatusHandStartd)
	c.pot = 0
	if c.ante > 0 {
		//前注
		c.doAnte()
	}
	c.publicCards = c.publicCards[:0]
	//洗牌
	c.poker.Reset()
	//下盲注
	c.smallBlind()
	c.bigBlind()
	//补盲
	if c.options.isPayToPlay {
		c.payToPlay()
	}
	//发牌（返回第一个行动的人）
	c.waitPause()
	firstAg := c.deal()
	//翻牌前下注
	users, showcard := c.preflop(firstAg)
	//如果只有一个人翻牌游戏结束
	if len(users) == 1 {
		c.simpleWin(users[0])
		return
	}
	//广播主边池内容
	c.sendPotsInfo(users, RoundPreFlop)
	//洗牌,并发送3张公共牌
	_, firstAg = c.dealPublicCards(3, RoundFlop)
	//未亮牌要下注
	if !showcard {
		//翻牌轮下注
		users, showcard = c.flopTurnRiver(firstAg, RoundFlop)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//广播主边池内容
	c.sendPotsInfo(users, RoundFlop)
	//已亮牌并且有保险开始保险逻辑
	if showcard && c.options.insuranceOpen {
		//等待买保险
		c.insuranceStart(users, RoundFlop)
	}
	//洗牌,并发送1张公共牌
	var cards []*Card
	cards, firstAg = c.dealPublicCards(1, RoundTurn)
	//已亮牌并且有保险开始保险计算
	if showcard && c.options.insuranceOpen {
		//保险计算结果
		c.insuranceEnd(cards[0], RoundFlop)
	}
	//未亮牌要下注
	if !showcard {
		//转牌轮下注
		users, showcard = c.flopTurnRiver(firstAg, RoundTurn)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//广播主边池内容
	c.sendPotsInfo(users, RoundTurn)
	//已亮牌并且有保险开始保险逻辑
	if showcard && c.options.insuranceOpen {
		//等待买保险
		c.insuranceStart(users, RoundTurn)
	}
	//洗牌,并发送1张公共牌
	cards, firstAg = c.dealPublicCards(1, RoundRiver)
	//已亮牌并且有保险开始保险计算
	if showcard && c.options.insuranceOpen {
		//保险计算结果
		c.insuranceEnd(cards[0], RoundTurn)
	}
	//未亮牌要下注
	if !showcard {
		//河牌轮下注
		users, _ = c.flopTurnRiver(firstAg, RoundRiver)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	c.sendPotsInfo(users, RoundRiver)
	//比牌计算结果
	c.complexWin(users)
}

//gameLoop 游戏逻辑
func (c *Holdem) gameLoop() {
	c.gameStatusCh = make(chan int8)
	defer close(c.gameStatusCh)
	v := <-c.gameStatusCh
	if v == GameStatusCancel {
		c.log.Debug("game cancel")
		//清理座位用户
		c.seatLock.Lock()
		for i, r := range c.players {
			r.gameInfo.resetForNextHand()
			c.log.Debug("user cancel stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
			c.standUp(i, r, StandUpGameEnd)
		}
		c.seatLock.Unlock()
		return
	}
	c.log.Debug("game start")
	c.options.recorder.GameStart(c.base())
	c.seatLock.Lock()
	for _, r := range c.roomers {
		r.recv.RoomerGameStart(c.id)
	}
	c.seatLock.Unlock()
	for {
		ok := c.buttonPosition()
		if !ok {
			c.log.Debug("players are not enough, wait")
			time.Sleep(c.options.waitForNotEnoughPlayers)
			continue
		}
		if c.nextSb > 0 {
			c.sb = uint(c.nextSb)
			c.nextSb = -1
		}
		if c.nextAnte >= 0 {
			c.ante = uint(c.nextAnte)
			c.nextAnte = -1
		}
		c.log.Debug("hand start")
		c.startHand()
		//清理座位用户
		waitforbuy := false
		bt := time.Now()
		c.seatLock.Lock()
		for i, r := range c.players {
			if c.options.autoStandUpMaxHand > 0 && r.auto && r.gameInfo.autoHandNum >= c.options.autoStandUpMaxHand {
				c.log.Debug("user stand up auto", zap.Int8("seat", i), zap.String("user", r.ID()))
				c.standUp(i, r, StandUpAutoExceedMaxTimes)
				continue
			}
			if r.gameInfo.chip == 0 && r.gameInfo.needStandUpReason == StandUpNone {
				waitforbuy = true
				c.delayStandUp(i, r, c.options.delayStandUpTimeout, StandUpNoChip)
				continue
			}
			if r.gameInfo.needStandUpReason != StandUpNone {
				c.log.Debug("user stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
				c.standUp(i, r, r.gameInfo.needStandUpReason)
			}
		}
		info := c.information()
		c.seatLock.Unlock()
		c.log.Debug("hand end")
		next := c.nextGame(info)
		if next {
			//清理座位用户
			c.log.Debug("hand end")
			if waitforbuy {
				wait := time.Since(bt) - c.options.delayStandUpTimeout - 500*time.Millisecond
				if wait > 0 {
					time.Sleep(wait)
				}
			}
			c.seatLock.Lock()
			for _, r := range c.players {
				r.gameInfo.resetForNextHand()
			}
			c.seatLock.Unlock()
			continue
		}
		c.statusChange(GameStatusComplete)
		//清理座位用户
		c.seatLock.Lock()
		for i, r := range c.players {
			r.gameInfo.resetForNextHand()
			c.log.Debug("user end stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
			c.standUp(i, r, StandUpGameEnd)
		}
		c.seatLock.Unlock()
		c.options.recorder.GameEnd(c.base())
		c.seatLock.Lock()
		for _, r := range c.roomers {
			r.recv.RoomerGameEnd(c.id)
		}
		c.seatLock.Unlock()
		c.log.Debug("game end")
		return
	}
}
