package holdem

import "sort"

type CardResult struct {
	Card     *Card
	Selected bool
	//CardIndex int //牌的位置 （0-1）手牌 （2-6）公共牌
}

type ShowCard struct {
	SeatNumber int8
	Cards      []*Card
}

func NewCardResult(card *Card, selected bool) *CardResult {
	return &CardResult{
		Card:     card,
		Selected: selected,
		//CardIndex: card.Num + card.Suit*15,
		//CardShow:  fmt.Sprintf("%s(%s)", card.NumString(), card.SuitString()),
	}
}

type Result struct {
	SeatNumber      int8
	Te              PlayType
	Num             uint
	Cards           []*CardResult
	HandValueType   HandValueType
	Chip            uint
	InsuranceResult map[Round]*InsuranceResult
}

//showDown 亮牌并计算获胜牌型，返回获胜的玩家和剩余的
func (c *Holdem) showDown(agents []*Agent) ([]*Agent, []*Agent) {
	th := make(map[int8]*HandValue)
	for _, r := range agents {
		r.gameInfo.calcHandValue(c.publicCards)
		th[r.gameInfo.seatNumber] = r.gameInfo.handValue
	}
	th = GetMaxHandValueFromTaggedHandValues(th)
	ret := make([]*Agent, 0)
	left := make([]*Agent, 0)
	for _, r := range agents {
		if _, ok := th[r.gameInfo.seatNumber]; ok {
			//c.log.Debug("winner", zap.Int8("seat", r.gameInfo.seatNumber), zap.Any("hv", r.gameInfo.handValue))
			ret = append(ret, r)
		} else {
			left = append(left, r)
		}
	}
	return ret, left
}

//calcWin 根据彩池和牌型分配奖励
func (c *Holdem) calcWin(urs []*Agent, pots []*Pot) (map[int8]*Result, []*Agent, []*Pot) {
	winners, leftUsers := c.showDown(urs)
	leftPots := make([]*Pot, 0)
	results := make(map[int8]*Result)
	for _, pot := range pots {
		result := make(map[int8]*Result)
		var l uint
		for _, w := range winners {
			if _, ok := pot.SeatNumber[w.gameInfo.seatNumber]; ok {
				l++
				result[w.gameInfo.seatNumber] = &Result{
					SeatNumber: w.gameInfo.seatNumber,
					Te:         w.gameInfo.te,
				}
			}
		}
		if l > 0 {
			award := pot.Num / l
			left := pot.Num - award*l
			for i, v := range result {
				v.Num = award
				result[i] = v
			}
			if left > 0 {
				u := c.button
				for {
					if v, ok := result[u.gameInfo.seatNumber]; ok {
						v.Num += left
						result[u.gameInfo.seatNumber] = v
						break
					}
				}
			}
		} else {
			leftPots = append(leftPots, pot)
		}
		for _, v := range result {
			if vv, ok := results[v.SeatNumber]; ok {
				vv.Num += v.Num
				results[v.SeatNumber] = vv
			} else {
				results[v.SeatNumber] = v
			}
		}
	}
	for len(leftPots) > 0 {
		var r2 map[int8]*Result
		r2, leftUsers, leftPots = c.calcWin(leftUsers, leftPots)
		for _, v := range r2 {
			results[v.SeatNumber] = v
		}
	}
	return results, leftUsers, leftPots
}

//calcPot 计算彩池(彩池边池，下注大小从小到大)
func (c *Holdem) calcPot(urs []*Agent) []*Pot {
	var mainPot uint
	u := c.button
	as := make([]*Agent, 0)
	users := make(map[int8]*Agent)
	for _, r := range urs {
		as = append(as, r)
		users[r.gameInfo.seatNumber] = r
	}
	ps := betSort(as)
	sort.Sort(ps)
	for {
		//不是最终玩家
		if _, ok := users[u.gameInfo.seatNumber]; !ok {
			mainPot += u.gameInfo.handBet
		} else {
			if u.gameInfo.status != ActionDefAllIn {
				mainPot += u.gameInfo.handBet
			}
		}
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	seats := make(map[int8]bool)
	for _, r := range ps {
		seats[r.gameInfo.seatNumber] = true
	}
	l := uint(len(ps))
	pots := make([]*Pot, 0)
	var lastAllIn uint
	for i, r := range ps {
		if r.gameInfo.status == ActionDefAllIn {
			if r.gameInfo.handBet == lastAllIn {
				l--
				delete(seats, r.gameInfo.seatNumber)
				continue
			}
			pot := &Pot{
				Num: l * (r.gameInfo.handBet - lastAllIn),
			}
			if i == 0 {
				pot.Num += mainPot
			}
			ss := make(map[int8]bool)
			for k, v := range seats {
				ss[k] = v
			}
			pot.SeatNumber = ss
			pots = append(pots, pot)
			l--
			lastAllIn = r.gameInfo.handBet
			delete(seats, r.gameInfo.seatNumber)
			continue
		}
		//mainPot += r.gameInfo.handBet
	}
	if len(pots) == 0 {
		pots = append(pots, &Pot{
			SeatNumber: seats,
			Num:        mainPot,
		})
	}
	return pots
}

type Pot struct {
	//SeatNumber 参与分配的座位号
	SeatNumber map[int8]bool
	//Num 池大小
	Num uint
}
