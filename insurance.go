package holdem

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type UserOut struct {
	Card        *Card
	HandValue   *HandValue
	CardResults []*CardResult
}

func (c *Holdem) insuranceStart(users []*Agent, round Round) {
	c.waitPause()
	//@TODO背保险
	c.log.Debug(round.String() + " buy insurance begin")
	cardsMap := make(map[int8][]*Card)
	us := make(map[int8]*Agent)
	for _, u := range users {
		cardsMap[u.gameInfo.seatNumber] = u.gameInfo.cards
		us[u.gameInfo.seatNumber] = u
	}
	pots := c.calcPot(users)
	currentHands, allNextHands := GetAllOuts(c.publicCards, cardsMap)
	leaderOuts := GetOuts(currentHands, allNextHands, pots)
	grp, _ := errgroup.WithContext(context.Background())
	ch := make(chan *InsuranceResult, len(users))
	c.insuranceUsers = make([]*Agent, 0)
	c.insuranceInformation = make(map[int8]map[int8][]*UserOut)
	c.waitPause()
	c.addWaitTime(c.options.insuranceWaitTimeout + delaySend)
	for leaderSeat, potOuts := range leaderOuts {
		u := us[leaderSeat]
		var insPot *Pot
		var o *Outs
		for pot, outs := range potOuts.Outs {
			if insPot == nil {
				insPot = pot
				o = outs
				continue
			}
			//池子人数最少的池子才买保险
			if len(pot.SeatNumber) > len(insPot.SeatNumber) {
				insPot = pot
				o = outs
			}
		}
		//没有赔率，无法购买
		odds, ok := c.options.insuranceOdds[o.Len]
		if !ok {
			u.recv.PlayerCanNotBuyInsurance(c.id, u.gameInfo.seatNumber, u.id, o.Len, round)
			continue
		}
		userOuts := make(map[int8][]*UserOut)
		for seat, hvs := range o.Detail {
			userOuts[seat] = make([]*UserOut, 0)
			for cd, hv := range hvs {
				cds := append(c.publicCards, u.gameInfo.cards...)
				cds = append(cds, cd)
				mp := hv.TaggingCards(cds)
				cr := make([]*CardResult, 0)
				for i := range cds {
					cr = append(cr, &CardResult{
						Card:     cds[i],
						Selected: mp[i],
					})
				}
				userOuts[seat] = append(userOuts[seat], &UserOut{
					Card:        cd,
					HandValue:   hv,
					CardResults: cr,
				})
			}
		}
		c.insuranceInformation[leaderSeat] = userOuts
		grp.Go(func() error {
			//稍微延迟告诉客户端可以买保险了
			time.AfterFunc(delaySend, func() {
				c.log.Debug("wait buy insurance", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", round.String()), zap.Int("outslen", o.Len))
				u.recv.PlayerCanBuyInsurance(c.id, u.gameInfo.seatNumber, c.id, o.Len, odds, userOuts, round)
				c.seatLock.Lock()
				for uid, rr := range c.roomers {
					if uid != u.ID() {
						rr.recv.RoomerGetWaitInsurance(c.id, u.gameInfo.seatNumber, u.id, c.options.insuranceWaitTimeout, round)
					}
				}
				c.seatLock.Unlock()
			})
			r, buy := u.waitBuyInsurance(o.Len, odds, userOuts, round, c.options.insuranceWaitTimeout)
			if r != nil {
				ch <- r
			}
			u.recv.PlayerBuyInsuranceSuccess(c.id, u.gameInfo.seatNumber, u.id, buy)
			c.seatLock.Lock()
			for uid, rr := range c.roomers {
				if uid != u.ID() {
					rr.recv.RoomerGetBuyInsurance(c.id, u.gameInfo.seatNumber, u.id, buy, round)
				}
			}
			c.seatLock.Unlock()
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
		var cost uint
		for _, v := range u.gameInfo.insurance {
			cost += v.Num
		}
		ins, ok := u.gameInfo.insurance[card.Value()]
		if ok {
			outsLen := c.insuranceResult[u.gameInfo.seatNumber][round].Outs
			c.insuranceResult[u.gameInfo.seatNumber][round].Earn = float64(ins.Num) * c.options.insuranceOdds[outsLen]
			c.options.recorder.InsureResult(c.base(), round, u.gameInfo.seatNumber, u.ID(), cost, c.insuranceResult[u.gameInfo.seatNumber][round].Earn)
		} else {
			c.options.recorder.InsureResult(c.base(), round, u.gameInfo.seatNumber, u.ID(), cost, 0)
		}
	}
}
