package example

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
)

type BetInfo struct {
	Chip       int //你还剩下多少筹码
	HandBet    int //本手你已经下了多少
	RoundBet   int //本轮你下了多少
	CurrentBet int //本轮轮到你现在下注额度多少
	MinRaise   int //最小加注金额
}

type ServerAction struct {
	Action  SA
	Action2 holdem.ActionDef
	Num     int
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
	nextGame := func(handCount uint) bool {
		if handCount >= count {
			s.complete = true
			return false
		}
		return true
	}
	s.h = holdem.NewHoldem(9, 100, 0, nextGame, log.With(zap.String("te", "server")))
	return s
}

func (c *Server) IsComplete() bool {
	return c.complete
}

func (c *Server) Connect(r *Robot) {
	id := rand.Intn(100)
	l := c.log.With(zap.String("te", "agent"))
	recv := &rec{
		ch:  r.InCh(),
		id:  fmt.Sprint(id),
		log: l,
	}
	agent := holdem.NewAgent(recv, &player{
		name: fmt.Sprintf("na-%d", id),
	}, l)
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
			c.hub.h.Join(c.agent)
		case RABringIn:
			c.agent.BringIn(o.Num)
		case RASeat:
			c.hub.h.Seated(int8(o.Num), c.agent)
		case RABet:
			c.agent.Bet(o.Bet)
		default:
			panic(fmt.Sprintf("unknown action %d", o.Action))
		}
	}
}
