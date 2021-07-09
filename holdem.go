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
	ID                  string
	Ante                uint
	SmallBlind          uint
	BigBlind            uint
	Metadata            map[string]interface{}
	HandNum             uint
	SBSeat              int8
	BBSeat              int8
	SeatCount           int8
	ButtonSeat          int8
	GameStatus          int8
	LimitAutoCheckTimes uint
	LimitAutoFoldTimes  uint
	WaitDeadline        time.Time
	LimitDelayTimes     uint
}

type HoldemState struct {
	*HoldemBase
	Seated      []*ShowUser
	EmptySeats  []int8
	Pot         uint
	PublicCards []*Card
	Onlines     uint
	Paused      bool
	Insurance   map[int8]map[int8][]*UserOut
}

type Holdem struct {
	id                   string                              //标识
	poker                *Poker                              //扑克
	handNum              uint                                //手数
	seatCount            int8                                //座位数量
	playerCount          int8                                //座位上用户数量
	players              map[int8]*Agent                     //座位上用户字典
	playingPlayerCount   int8                                //当前游戏玩家数量（座位上游戏的）
	roomers              map[string]*Agent                   //参与游戏的玩家（包括旁观）
	buttonSeat           int8                                //庄家座位号
	sbSeat               int8                                //小盲座位号
	bbSeat               int8                                //大盲座位号
	payToPlayMap         map[int8]PlayType                   //补牌的规则
	button               *Agent                              //庄家玩家
	waitBetTimeout       time.Duration                       //等待下注的超时时间
	seatLock             sync.Mutex                          //玩家锁
	gameStartedLock      int32                               //是否开始原子锁
	gameStatusCh         chan int8                           //开始通道
	handStartInfo        *StartNewHandInfo                   //当前一手开局信息
	sb                   uint                                //小盲
	nextSb               int                                 //即将修改的小盲
	ante                 uint                                //前注
	nextAnte             int                                 //即将修改的前注
	pot                  uint                                //彩池
	roundBet             uint                                //当前轮下注额
	minRaise             uint                                //最小加注量
	publicCards          []*Card                             //公共牌
	log                  *zap.Logger                         //日志
	nextGame             func(*HoldemState) bool             //是否继续下一轮的回调函数和等待下一手时间(当前手数) - 内部可以用各种条件来判断是否继续
	insuranceInformation map[int8]map[int8][]*UserOut        //当前保险Outs 可以买保险的Seat:对应Outs的座位号:[]outs
	insuranceResult      map[int8]map[Round]*InsuranceResult //保险结果
	insuranceUsers       []*Agent                            //参与保险的玩家
	waitDeadline         time.Time                           //等待的截止时间
	paused               bool                                //暂停
	pauseCh              chan bool                           //暂停通道
	options              *extOptions                         //额外配置
}

func NewHoldem(
	id string,
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
		recorder:                &NopRecorder{},
		isPayToPlay:             false,
		medadata:                make(map[string]interface{}),
		waitForNotEnoughPlayers: 10 * time.Second,
		minPlayers:              2,
		limitDelayTimes:         2,
		limitAutoCheckTimes:     4,
		limitAutoFoldTimes:      3,
	}
	for _, o := range ops {
		o.apply(exts)
	}
	if exts.minPlayers > sc {
		exts.minPlayers = sc
	}
	if exts.minPlayers < 2 {
		exts.minPlayers = 2
	}
	if exts.autoStart {
		if exts.autoMinPlayers > sc {
			exts.autoMinPlayers = sc
		}
		if exts.autoMinPlayers < 2 {
			exts.autoMinPlayers = 2
		}
	}
	h := &Holdem{
		id:             id,
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
		oldRs.recv.PlayerJoinSuccess(c.id, rs.ID(), c.information(oldRs))
		return
	}
	c.roomers[rs.ID()] = rs
	for uid, r := range c.roomers {
		if uid != rs.ID() {
			r.recv.RoomerJoin(c.id, rs.ID())
		}
	}
	rs.recv.PlayerJoinSuccess(c.id, rs.ID(), c.information(rs))
}

//leave 离开
func (c *Holdem) leave(rs *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	delete(c.roomers, rs.ID())
	rs.recv.PlayerLeaveSuccess(c.id, rs.ID())
	for uid, r := range c.roomers {
		if uid != rs.ID() {
			r.recv.RoomerLeave(c.id, rs.ID())
		}
	}
}

//Seated 坐下
func (c *Holdem) seated(i int8, r *Agent) {
	if r.gameInfo == nil || r.gameInfo.chip < c.ante+c.sb*2 {
		r.recv.ErrorOccur(c.id, ErrCodeNoChip, errNoChip)
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
			r.recv.ErrorOccur(c.id, ErrCodeTableIsFull, errTableIsFull)
			return
		}
	} else {
		if c.players[i] != nil {
			c.seatLock.Unlock()
			r.recv.ErrorOccur(c.id, ErrCodeSeatTaken, errSeatTaken)
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
	r.recv.PlayerSeatedSuccess(c.id, i, r.id, r.gameInfo.te)
	//通知其他人
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerSeated(c.id, i, r.id, r.gameInfo.te)
		}
	}
	info := c.information()
	c.seatLock.Unlock()
	if c.status() == GameStatusNotStart && c.options.autoStart && c.playerCount >= c.options.autoMinPlayers {
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
	r.recv.PlayerKeepSeat(c.id, i, r.id, tm)
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerKeepSeat(c.id, i, r.id, tm)
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
	r.recv.PlayerStandUp(c.id, i, r.id, reason)
	//通知其他人
	for uid, rr := range c.roomers {
		if uid != r.ID() {
			rr.recv.RoomerStandUp(c.id, i, r.id, reason)
		}
	}
}

//status 状态
func (c *Holdem) status() int8 {
	v := atomic.LoadInt32(&c.gameStartedLock)
	return int8(v)
}

//statusChange 状态改变
func (c *Holdem) statusChange(status int8) {
	atomic.StoreInt32(&c.gameStartedLock, int32(status))
}

//base 基础信息（无锁)
func (c *Holdem) base() *HoldemBase {
	return &HoldemBase{
		ID:                  c.id,
		Ante:                c.ante,
		SmallBlind:          c.sb,
		SBSeat:              c.sbSeat,
		BigBlind:            c.sb * 2,
		BBSeat:              c.bbSeat,
		SeatCount:           c.seatCount,
		ButtonSeat:          c.buttonSeat,
		GameStatus:          c.status(),
		HandNum:             c.handNum,
		Metadata:            c.options.medadata,
		WaitDeadline:        c.waitDeadline,
		LimitAutoCheckTimes: c.options.limitAutoCheckTimes,
		LimitAutoFoldTimes:  c.options.limitAutoFoldTimes,
		LimitDelayTimes:     c.options.limitDelayTimes,
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
		Insurance:   c.insuranceInformation,
		Paused:      c.paused,
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
					p.recv.PlayerCanPayToPlay(c.id, seat, p.id)
				}
				continue
			}
			p.prevAgent = cur
			cur.nextAgent = p
			cur = p
		}
	}
	newButton.prevAgent = cur
	cur.nextAgent = newButton
	//坐着的人比约定人数少 不开始比赛也不轮转
	if playerCount < c.options.minPlayers {
		c.log.Debug("button position end(false)", zap.Int8("minplayers", c.options.minPlayers), zap.Int8("valid seat count", playerCount), zap.Int8("seat count", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
		return false
	}
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
	//所有能玩的人数小于最小人数，不开始，但是轮转
	if c.playingPlayerCount < c.options.minPlayers {
		c.log.Debug("button position end(false)", zap.Int8("minplayers", c.options.minPlayers), zap.Int8("seat count", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
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

//addWaitTime 游戏等待时间计算（内部使用)
func (c *Holdem) addWaitTime(dur time.Duration) {
	c.waitDeadline = time.Now().Add(dur)
}

//checkRoundComplete 判断是否叫注轮结束（内部使用)
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

//getNextOpAgent 计算下一个操作者
func (c *Holdem) getNextOpAgent(u *Agent) *Agent {
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

//sendPotsInfo 发送主边池信息
func (c *Holdem) sendPotsInfo(users []*Agent, round Round) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	pots := c.calcPot(users)
	for _, r := range c.roomers {
		r.recv.RoomerGamePots(c.id, pots, round)
	}
}

//autoOp 托管
func (c *Holdem) autoOp(r *Agent, open bool) {
	r.auto = open
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	if !open {
		r.gameInfo.autoCheckTimes = 0
		r.gameInfo.autoFoldTimes = 0
	}
	for _, r := range c.roomers {
		r.recv.RoomerAutoOp(c.id, r.gameInfo.seatNumber, r.id, open)
	}
}

//exceedOpTime 延时
func (c *Holdem) exceedOpTime(r *Agent, tm time.Duration) {
	if r.gameInfo.delayTimes > c.options.limitDelayTimes {
		r.recv.ErrorOccur(c.id, ErrCodeExceedTimeOverTimes, errExceedTimeOverTimes)
		return
	}
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	r.recv.PlayerExceedTimeSuccess(c.id, r.gameInfo.seatNumber, r.id, int8(r.gameInfo.delayTimes), tm)
	for _, rr := range c.roomers {
		if rr.id != r.id {
			rr.recv.RoomerExceedTime(c.id, r.gameInfo.seatNumber, r.id, int8(r.gameInfo.delayTimes), tm)
		}
	}
}

//waitPause 等暂停结束（内部使用)
func (c *Holdem) waitPause() {
	if c.paused {
		<-c.pauseCh
		c.log.Debug("resume")
		c.seatLock.Lock()
		defer c.seatLock.Unlock()
		for _, rr := range c.roomers {
			rr.recv.RoomerGamePauseResume(c.id, false)
		}
	}
}

//fakeAgent 占位Agent(占座不玩牌/已经离开/等等)
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
