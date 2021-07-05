package example

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
)

type BetInfo struct {
	Chip       uint //你还剩下多少筹码
	HandBet    uint //本手你已经下了多少
	RoundBet   uint //本轮你下了多少
	CurrentBet uint //本轮轮到你现在下注额度多少
	MinRaise   uint //最小加注金额
}

type ServerAction struct {
	Action  SA
	Action2 holdem.ActionDef
	Num     uint
	Seat    int8
	Payload []byte
	BetInfo *BetInfo
}

type Server struct {
	h        *holdem.Holdem
	log      *zap.Logger
	from, to time.Time
	hands    uint
	complete bool
}

type agentWrapper struct {
	r     *Robot
	agent *holdem.Agent
	hub   *Server
}

func NewServer(from time.Time, to time.Time, count uint, log *zap.Logger) *Server {
	s := &Server{
		log:      log,
		from:     from,
		to:       to,
		hands:    count,
		complete: false,
	}
	nextGame := func(h *holdem.Holdem) (bool, time.Duration) {
		if h.State().HandNum >= count {
			s.complete = true
			return false, 0
		}
		return true, 10 * time.Second
	}
	mp := make(map[int]float64)
	for i := 1; i < 30; i++ {
		mp[i] = rand.Float64() * 100
	}
	s.h = holdem.NewHoldem(9, 100, 20*time.Second, nextGame, log.With(zap.String("te", "server")), holdem.OptionPayToPlay(), holdem.OptionInsurance(mp, 10*time.Second), holdem.OptionAutoStart(2))
	return s
}

func (c *Server) IsComplete() bool {
	return c.complete
}

func (c *Server) Connect(r *Robot) {
	l := c.log.With(zap.String("te", "agent"))
	recv := &rec{
		ch:  r.InCh(),
		id:  r.ID,
		log: l,
	}
	agent := holdem.NewAgent(recv, l)
	a := &agentWrapper{
		r:     r,
		agent: agent,
		hub:   c,
	}
	go a.read()
}

func (c *agentWrapper) read() {
	for o := range c.r.OutCh() {
		switch o.Action {
		case RAJoin:
			c.agent.Join(c.hub.h)
		case RABringIn:
			c.agent.BringIn(o.Num)
		case RASeat:
			//c.agent.Seated(int8(o.Num))
			c.agent.Seated()
		case RAInfo:
			c.agent.Info()
		case RABet:
			c.agent.Bet(o.Bet)
		case RAStandUp:
			c.agent.StandUp()
		default:
			panic(fmt.Sprintf("unknown action %d", o.Action))
		}
	}
}
