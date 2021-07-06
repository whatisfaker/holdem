package holdem

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

//StartNewHandInfo 新一手新自动操作信息
type StartNewHandInfo struct {
	//AnteAllIns 前注AllIn的座位号数组
	AnteAllIns []int8
	//SB 小盲下注信息(nil/SB/AllIn)
	SB     *Bet
	SBSeat int8
	//BB 小盲下注信息(nil/BB/AllIn)
	BB     *Bet
	BBSeat int8
	//PayToPlay 做了补盲的用户数组
	PayToPlay []int8
}

type HoldemBase struct {
	Ante       uint
	SmallBlind uint
	BigBlind   uint
	Metadata   map[string]interface{}
	HandNum    uint
	SBSeat     int8
	BBSeat     int8
	SeatCount  int8
	ButtonSeat int8
	GameStatus int8
}

type HoldemState struct {
	*HoldemBase
	Seated      []*ShowUser
	EmptySeats  []int8
	Pot         uint
	PublicCards []*Card
	Onlines     uint
}

type Holdem struct {
	poker              *Poker                              //扑克
	handNum            uint                                //手数
	seatCount          int8                                //座位数量
	playerCount        int8                                //座位上用户数量
	players            map[int8]*Agent                     //座位上用户字典
	playingPlayerCount int8                                //当前游戏玩家数量（座位上游戏的）
	roomers            map[string]*Agent                   //参与游戏的玩家（包括旁观）
	buttonSeat         int8                                //庄家座位号
	sbSeat             int8                                //小盲座位号
	bbSeat             int8                                //大盲座位号
	payToPlayMap       map[int8]PlayType                   //补牌的规则
	button             *Agent                              //庄家玩家
	waitBetTimeout     time.Duration                       //等待下注的超时时间
	seatLock           sync.Mutex                          //玩家锁
	gameStartedLock    int32                               //是否开始原子锁
	gameStatusCh       chan int8                           //开始通道
	handStartInfo      *StartNewHandInfo                   //当前一手开局信息
	sb                 uint                                //小盲
	nextSb             int                                 //即将修改的小盲
	ante               uint                                //前注
	nextAnte           int                                 //即将修改的前注
	pot                uint                                //彩池
	roundBet           uint                                //当前轮下注额
	minRaise           uint                                //最小加注量
	publicCards        []*Card                             //公共牌
	log                *zap.Logger                         //日志
	nextGame           func(*HoldemState) bool             //是否继续下一轮的回调函数和等待下一手时间(当前手数) - 内部可以用各种条件来判断是否继续
	insuranceResult    map[int8]map[Round]*InsuranceResult //保险结果
	insuranceUsers     []*Agent                            //参与保险的玩家
	options            *extOptions                         //额外配置
}

func NewHoldem(
	sc int8, //座位数
	sb uint, //小盲
	waitBetTimeout time.Duration, //等待下注超时时间
	nextGame func(*HoldemState) bool, //是否继续下一手判断/等待时间
	log *zap.Logger, //日志
	ops ...HoldemOption,
) *Holdem {
	if nextGame == nil {
		nextGame = func(*HoldemState) bool {
			return true
		}
	}
	payMap := make(map[int8]PlayType)
	var i int8
	for i = 1; i <= sc; i++ {
		payMap[i] = PlayTypeNormal
	}
	exts := &extOptions{
		insuranceOpen:           false,
		recorder:                newNopRecorder(),
		isPayToPlay:             false,
		medadata:                make(map[string]interface{}),
		waitForNotEnoughPlayers: 10 * time.Second,
	}
	for _, o := range ops {
		o.apply(exts)
	}
	if exts.autoStart {
		if exts.minPlayers > sc {
			exts.minPlayers = sc
		}
		if exts.minPlayers < 2 {
			exts.minPlayers = 2
		}
	}
	h := &Holdem{
		poker:          NewPoker(),
		players:        make(map[int8]*Agent),
		roomers:        make(map[string]*Agent),
		publicCards:    make([]*Card, 0, 5),
		waitBetTimeout: waitBetTimeout,
		seatCount:      sc,
		sb:             sb,
		nextSb:         -1,
		ante:           exts.ante,
		nextAnte:       -1,
		log:            log,
		nextGame:       nextGame,
		payToPlayMap:   payMap,
		options:        exts,
	}
	go h.gameLoop()
	return h
}

//Join 加入游戏,并没有坐下(重新进入逻辑)
func (c *Holdem) join(rs *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	oldRs, ok := c.roomers[rs.ID()]
	if ok {
		oldRs.replace(rs)
		c.roomers[rs.ID()] = oldRs
		oldRs.recv.PlayerJoinSuccess(rs.ID(), c.information(oldRs))
		return
	}
	c.roomers[rs.ID()] = rs
	for uid, r := range c.roomers {
		if uid != rs.ID() {
			r.recv.RoomerJoin(rs.ID())
		}
	}
	rs.recv.PlayerJoinSuccess(rs.ID(), c.information(rs))
}

//leave 离开
func (c *Holdem) leave(rs *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	delete(c.roomers, rs.ID())
	rs.recv.PlayerLeaveSuccess(rs.ID())
	for uid, r := range c.roomers {
		if uid != rs.ID() {
			r.recv.RoomerLeave(rs.ID())
		}
	}
}

//Seated 坐下
func (c *Holdem) seated(i int8, r *Agent) {
	if r.gameInfo == nil || r.gameInfo.chip < c.ante+c.sb*2 {
		r.ErrorOccur(ErrCodeNoChip, errNoChip)
		return
	}
	c.seatLock.Lock()
	//自动找座
	if i == 0 {
		var idx int8 = 1
		for ; idx <= c.seatCount; idx++ {
			if _, ok := c.players[idx]; !ok {
				i = idx
			}
		}
		if i == 0 {
			c.seatLock.Unlock()
			r.ErrorOccur(ErrCodeTableIsFull, errTableIsFull)
			return
		}
	} else {
		if c.players[i] != nil {
			c.seatLock.Unlock()
			r.ErrorOccur(ErrCodeSeatTaken, errSeatTaken)
			return
		}
	}
	r.gameInfo.seatNumber = i
	r.gameInfo.te = PlayTypeNormal
	c.players[i] = r
	c.playerCount++
	//开启补盲
	r.gameInfo.te = c.payToPlayMap[i]
	c.log.Debug("user seated", zap.Int8("seat", i), zap.String("na", c.players[i].ID()), zap.Int8("te", int8(r.gameInfo.te)))
	//通知自己坐下了
	r.recv.PlayerSeatedSuccess(i, r.gameInfo.te)
	//通知其他人
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerSeated(i, r.ID(), r.gameInfo.te)
		}
	}
	info := c.information()
	c.seatLock.Unlock()
	if c.status() == GameStatusNotStart && c.options.autoStart && c.playerCount >= c.options.minPlayers {
		if ok := c.nextGame(info); ok {
			c.Start()
		}
	}
}

//directStandUp 不用等待本手结束直接站起来
func (c *Holdem) directStandUp(i int8, r *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	c.log.Debug("user direct stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
	c.standUp(i, r, StandUpAction)
}

func (c *Holdem) delayStandUp(i int8, r *Agent, tm time.Duration, reason int8) {
	c.log.Debug("delay stand up", zap.Int8("seat", i), zap.String("user", r.ID()), zap.Duration("dur", tm))
	r.recv.PlayerKeepSeat(i, tm)
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerKeepSeat(i, tm)
		}
	}
	time.AfterFunc(tm, func() {
		//去了其他游戏
		if r.h != c {
			return
		}
		//已经自行站起来
		if r.gameInfo == nil {
			return
		}
		//游戏已经结束
		if c.status() == GameStatusComplete || c.status() == GameStatusCancel {
			return
		}
		//还是空筹码
		if r.gameInfo.chip == 0 && r.gameInfo.seatNumber == i {
			c.log.Debug("less chip auto stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
			c.seatLock.Lock()
			c.standUp(i, r, reason)
			c.seatLock.Unlock()
		}
	})
}

//standUp 站起来
func (c *Holdem) standUp(i int8, r *Agent, reason int8) {
	//c.log.Debug("standup", zap.Int8("seat", i), zap.Bool("fake", r.fake), zap.String("na", c.players[i].ID()), zap.Int8("te", int8(r.gameInfo.te)))
	r.gameInfo = nil
	delete(c.players, i)
	c.playerCount--
	//通知自己站起来了
	r.recv.PlayerStandUp(i, reason)
	//通知其他人
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerStandUp(i, r.ID(), reason)
		}
	}
}

//Status 状态
func (c *Holdem) status() int8 {
	v := atomic.LoadInt32(&c.gameStartedLock)
	return int8(v)
}

func (c *Holdem) base() *HoldemBase {
	return &HoldemBase{
		Ante:       c.ante,
		SmallBlind: c.sb,
		SBSeat:     c.sbSeat,
		BigBlind:   c.sb * 2,
		BBSeat:     c.bbSeat,
		SeatCount:  c.seatCount,
		ButtonSeat: c.buttonSeat,
		GameStatus: c.status(),
		HandNum:    c.handNum,
		Metadata:   c.options.medadata,
	}
}

//Information 游戏信息
func (c *Holdem) information(rs ...*Agent) *HoldemState {
	rMap := make(map[*Agent]bool)
	for _, r := range rs {
		rMap[r] = true
	}
	players := make([]*ShowUser, 0)
	var s int8
	emptySeats := make([]int8, 0)
	for s = 1; s <= c.seatCount; s++ {
		p, ok := c.players[s]
		if ok {
			_, ok2 := rMap[p]
			players = append(players, p.displayUser(ok2))
		} else {
			emptySeats = append(emptySeats, s)
		}
	}
	return &HoldemState{
		HoldemBase:  c.base(),
		Seated:      players,
		EmptySeats:  emptySeats,
		Pot:         c.pot,
		PublicCards: c.publicCards,
		Onlines:     uint(len(c.roomers)),
	}
}

//buttonPosition 决定按钮位置
func (c *Holdem) buttonPosition() bool {
	c.log.Debug("button position begin", zap.Int8("seat count", c.playerCount))
	var cur *Agent
	var i, buIdx int8
	if c.handNum == 0 {
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		buIdx = int8(rd.Intn(int(c.seatCount))) + 1
	} else {
		//庄位移动
		buIdx = c.sbSeat
	}
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	c.playingPlayerCount = 0
	payMap := make(map[int8]PlayType)
	var newButton *Agent
	var newButtonSeat int8
	var playerCount int8
	for i = 0; i < c.seatCount; i++ {
		seat := ((i + buIdx - 1) % c.seatCount) + 1
		if c.options.isPayToPlay {
			payMap[seat] = PlayTypeNeedPayToPlay
		} else {
			payMap[seat] = PlayTypeNormal
		}
		p, ok := c.players[seat]
		if ok {
			//无筹码留座的直接跳过
			if p.gameInfo.chip == 0 {
				continue
			}
			playerCount++
			//不需要补盲(也不再禁止位置)
			if !c.options.isPayToPlay && p.gameInfo.te != PlayTypeDisable {
				p.gameInfo.te = PlayTypeNormal
			}
			if p.gameInfo.te == PlayTypeNormal || p.gameInfo.te == PlayTypeAgreePayToPlay {
				c.playingPlayerCount++
			}
			if cur == nil {
				newButton = p
				newButtonSeat = seat
				cur = p
				if p.gameInfo.te == PlayTypeDisable {
					p.gameInfo.te = PlayTypeNeedPayToPlay
					//通知玩家可以补盲了
					p.recv.PlayerCanPayToPlay(seat)
				}
				continue
			}
			p.prevAgent = cur
			cur.nextAgent = p
			cur = p
		}
	}
	if playerCount <= 1 {
		c.log.Debug("button position end(false)", zap.Int8("valid seat count", playerCount), zap.Int8("seat count", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
		return false
	}
	newButton.prevAgent = cur
	cur.nextAgent = newButton
	newSBSeat := newButton.nextAgent.gameInfo.seatNumber
	newBBSeat := newButton.nextAgent.nextAgent.gameInfo.seatNumber
	//BB位可以脱离补盲状态
	if newButton.nextAgent.nextAgent.gameInfo.te == PlayTypeNeedPayToPlay || newButton.nextAgent.nextAgent.gameInfo.te == PlayTypeDisable {
		newButton.nextAgent.nextAgent.gameInfo.te = PlayTypeNormal
		c.playingPlayerCount++
	}
	//bu到sb之间的位置都是禁止位（不发手牌)
	if newButtonSeat > newSBSeat {
		for i := newButtonSeat + 1; i <= newSBSeat+c.seatCount; i++ {
			payMap[i%c.seatCount] = PlayTypeDisable
		}
	} else {
		for i := newButtonSeat + 1; i <= newSBSeat; i++ {
			payMap[i%c.seatCount] = PlayTypeDisable
		}
	}
	payMap[newBBSeat] = PlayTypeDisable
	u := newButton
	//用fakeAgent替换掉坐着但不发牌的人
	for {
		if u.gameInfo.te == PlayTypeNeedPayToPlay || u.gameInfo.te == PlayTypeDisable {
			u2 := fakeAgent(u)
			if u == newButton {
				newButton = u2
			}
			u = u2
		}
		u = u.nextAgent
		if u == newButton {
			break
		}
	}
	c.button = newButton
	c.buttonSeat = newButtonSeat
	c.sbSeat = newSBSeat
	c.bbSeat = newBBSeat
	c.payToPlayMap = payMap
	if c.playingPlayerCount <= 1 {
		c.log.Debug("button position end(false)", zap.Int8("seat count", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
		return false
	}
	c.handStartInfo = &StartNewHandInfo{
		AnteAllIns: []int8{},
		PayToPlay:  []int8{},
		SBSeat:     c.sbSeat,
		BBSeat:     c.bbSeat,
	}
	u = c.button
	for {
		if !u.fake {
			u.gameInfo.handNum++
			if !u.auto {
				u.gameInfo.autoHandNum = 0
			} else {
				u.gameInfo.autoHandNum++
			}
		}
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	c.handNum++
	c.options.recorder.HandBegin(c.information())
	c.log.Debug("button position end(true)", zap.Int8("buseat", c.buttonSeat), zap.String("buuser", c.button.ID()), zap.Int8("players", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
	return true
}

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

//preflop 翻牌前叫注
func (c *Holdem) preflop(op *Agent) ([]*Agent, bool) {
	c.roundBet = c.sb * 2
	c.minRaise = c.sb * 2
	u := op
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	c.log.Debug(RoundPreFlop.String()+" bet begin", zap.Int8("pc", c.playingPlayerCount), zap.Int8("sseat", u.gameInfo.seatNumber), zap.String("suser", u.ID()))
	for u != nil {
		c.log.Debug("wait bet", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", RoundPreFlop.String()))
		bet := u.waitBet(c.roundBet, c.minRaise, RoundPreFlop, c.waitBetTimeout)
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
			next = c.getNextOperator(u)
			op = newOperator(next, c.roundBet, c.minRaise)
		}
		thisAgent := u
		//稍微延迟告诉客户端可以下注
		time.AfterFunc(delaySend, func() {
			thisAgent.recv.PlayerActionSuccess(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op)
			c.seatLock.Lock()
			defer c.seatLock.Unlock()
			for uid, r := range c.roomers {
				if uid != thisAgent.ID() {
					r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op, r == next)
				}
			}
		})
		u = next
	}
	//等500ms
	time.Sleep(2 * delaySend)
	if showcard {
		scs := make([]*ShowCard, 0)
		for _, v := range unfoldUsers {
			scs = append(scs, &ShowCard{
				SeatNumber: v.gameInfo.seatNumber,
				Cards:      v.gameInfo.cards,
			})
		}
		c.seatLock.Lock()
		for _, r := range c.roomers {
			r.recv.RoomerGetShowCards(scs)
		}
		c.seatLock.Unlock()
	}
	c.log.Debug(RoundPreFlop.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//flopTurnRiver 叫注（轮描述）
func (c *Holdem) flopTurnRiver(u *Agent, round Round) ([]*Agent, bool) {
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
		c.log.Debug("wait bet", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", round.String()))
		bet := u.waitBet(c.roundBet, c.minRaise, round, c.waitBetTimeout)
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
			next = c.getNextOperator(u)
			op = newOperator(next, c.roundBet, c.minRaise)
		}
		//稍微延迟告诉客户端可以下注
		thisAgent := u
		time.AfterFunc(delaySend, func() {
			thisAgent.recv.PlayerActionSuccess(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op)
			c.seatLock.Lock()
			defer c.seatLock.Unlock()
			for uid, r := range c.roomers {
				if uid != thisAgent.ID() {
					r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op, r == next)
				}
			}
		})
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
			r.recv.RoomerGetShowCards(scs)
		}
		c.seatLock.Unlock()
	}
	c.log.Debug(round.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//checkRoundComplete 判断是否叫注轮结束
func (c *Holdem) checkRoundComplete() (bool, []*Agent, bool) {
	u := c.button
	users := make([]*Agent, 0)
	allInCount := 0
	for {
		//已盖牌/未发牌玩家跳过
		if u.fake {
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		//All in 作为未盖牌跟进
		if u.gameInfo.status == ActionDefAllIn {
			users = append(users, u)
			allInCount++
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		//本轮下注不等于本轮的注,直接返回未结束
		if c.roundBet > 0 && u.gameInfo.roundBet != c.roundBet {
			return false, nil, false
		}
		users = append(users, u)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	//还没有下注但是玩家大于1,还未结束
	if c.roundBet == 0 && len(users) > 1 {
		return false, nil, false
	}
	return true, users, allInCount > 0 && len(users) > 1 && allInCount >= len(users)-1
}

func (c *Holdem) getNextOperator(u *Agent) *Agent {
	first := u
	u = u.nextAgent
	for {
		if u == first {
			return nil
		}
		//已盖牌/未发牌跳过
		if u.fake {
			u = u.nextAgent
		} else if u.gameInfo.status == ActionDefAllIn {
			//All In的也跳过
			u = u.nextAgent
		} else {
			return u
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
	firstAg := c.getNextOperator(c.button.nextAgent.nextAgent)
	op := newOperator(firstAg, 2*c.sb, 2*c.sb)
	//延迟告诉客户端,让服务器可以提前开启等待bet的channel(preflop::waitBet),以免请求早于接收通道开启
	time.AfterFunc(delaySend, func() {
		c.seatLock.Lock()
		defer c.seatLock.Unlock()
		first := c.button.nextAgent
		cur = first
		i := 0
		seats := make([]int8, 0)
		for {
			//在座，但是不能发牌
			if cur.fake {
				cur = cur.nextAgent
			} else {
				cur.gameInfo.cards = cards[i]
				cur.recv.PlayerGetCard(cur.gameInfo.seatNumber, cards[i], seats, int8(cnt), c.handStartInfo, op, cur == firstAg)
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
				r.recv.RoomerGetCard(seats, int8(cnt), c.handStartInfo, op)
			}
		}
	})
	c.log.Debug("deal end")
	return firstAg
}

//dealPublicCards 发公共牌
func (c *Holdem) dealPublicCards(n int, round Round) ([]*Card, *Agent) {
	c.log.Debug("deal public cards(start)", zap.Int("cards_count", n))
	//洗牌
	_, _ = c.poker.GetCards(1)
	cards, _ := c.poker.GetCards(n)
	c.publicCards = append(c.publicCards, cards...)
	firstAg := c.getNextOperator(c.button)
	firstOp := newOperator(firstAg, 0, 2*c.sb)
	//延迟告诉客户端,让服务器可以提前开启等待bet的channel(flopTurnRiver::waitBet),以免请求早于接收通道开启
	time.AfterFunc(delaySend, func() {
		c.seatLock.Lock()
		defer c.seatLock.Unlock()
		for _, r := range c.roomers {
			r.recv.RoomerGetPublicCard(cards, firstOp, firstAg == r)
		}
	})
	c.log.Debug("deal public cards(end)", zap.Int("cards_count", n))
	return cards, firstAg
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
		r.recv.RoomerGetResult(ret)
	}
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
		r.recv.RoomerGetResult(ret)
	}
	c.options.recorder.HandEnd(c.information(), ret)
	c.seatLock.Unlock()
	c.log.Debug("swin", zap.Int8("seat", agent.gameInfo.seatNumber), zap.String("user", agent.ID()), zap.Any("result", ret))
}

//StartHand 开始新的一手
func (c *Holdem) startHand() {
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
	firstAg := c.deal()
	//翻牌前下注
	users, showcard := c.preflop(firstAg)
	//如果只有一个人翻牌游戏结束
	if len(users) == 1 {
		c.simpleWin(users[0])
		return
	}
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
	//比牌计算结果
	c.complexWin(users)
}

func fakeAgent(p *Agent) *Agent {
	ret := NewAgent(p.recv, p.id, p.log)
	//状态重置
	p.gameInfo.status = ActionDefNone
	ret.gameInfo = p.gameInfo
	ret.auto = p.auto
	ret.fake = true
	ret.prevAgent = p.prevAgent
	ret.nextAgent = p.nextAgent
	p.prevAgent = nil
	p.nextAgent = nil
	ret.prevAgent.nextAgent = ret
	ret.nextAgent.prevAgent = ret
	return ret
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
		atomic.StoreInt32(&c.gameStartedLock, int32(GameStatusComplete))
		//清理座位用户
		c.seatLock.Lock()
		for i, r := range c.players {
			r.gameInfo.resetForNextHand()
			c.log.Debug("user end stand up", zap.Int8("seat", i), zap.String("user", r.ID()))
			c.standUp(i, r, StandUpGameEnd)
		}
		c.seatLock.Unlock()
		c.options.recorder.GameEnd(c.base())
		c.log.Debug("game end")
		return
	}
}
