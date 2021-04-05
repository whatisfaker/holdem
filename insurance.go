package holdem

import (
	"context"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func (c *Holdem) insuranceStart(users []*Agent, round Round) {
	c.log.Debug(round.String() + " buy insurance begin")
	cardsMap := make(map[int8][]*Card)
	us := make(map[int8]*Agent)
	for _, u := range users {
		cardsMap[u.gameInfo.seatNumber] = u.gameInfo.cards
		us[u.gameInfo.seatNumber] = u
	}
	_, groups := c.calcPot(users)
	currentHands, allNextHands := GetAllOuts(c.publicCards, cardsMap)
	outs := GetOuts(currentHands, allNextHands, groups)
	grp, _ := errgroup.WithContext(context.Background())
	ch := make(chan *InsuranceResult, len(users))
	c.insuranceUsers = make([]*Agent, 0)
	for _, out := range outs {
		u := us[out.TargetSeat]
		o := out
		odds, ok := c.insuranceOdds[o.Len]
		if !ok {
			u.recv.PlayerCanNotBuyInsurance(u.gameInfo.seatNumber, out.Len, round)
			continue
		}
		grp.Go(func() error {
			r := u.waitBuyInsurance(o.Len, odds, o.Detail, round, c.insuranceWaitTimeout)
			if r != nil {
				ch <- r
			}
			return nil
		})
	}
	_ = grp.Wait()
	close(ch)
	for r := range ch {
		res, ok := c.insuranceResult[r.SeatNumber]
		if !ok {
			res = make(map[Round]*InsuranceResult)
		}
		c.insuranceUsers = append(c.insuranceUsers, us[r.SeatNumber])
		res[round] = r
		c.insuranceResult[r.SeatNumber] = res
	}
	c.log.Debug(round.String()+" buy insurance end", zap.Int("len", len(c.insuranceUsers)))
}

func (c *Holdem) insuranceEnd(card *Card, round Round) {
	for _, u := range c.insuranceUsers {
		ins, ok := u.gameInfo.insurance[card.Value()]
		if ok {
			outsLen := c.insuranceResult[u.gameInfo.seatNumber][round].Outs
			c.insuranceResult[u.gameInfo.seatNumber][round].Earn = float64(ins.Num) * c.insuranceOdds[outsLen]
		}
	}
}
