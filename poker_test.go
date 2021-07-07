package holdem

import (
	"testing"
	"time"

	"gopkg.in/stretchr/testify.v1/assert"
)

func TestNewPoker(t *testing.T) {
	a := NewPoker()
	i := 2
	for i > 0 {
		cs, err := a.GetCards(5)
		if err != nil {
			t.Error(err)
			return
		}
		hands := make(map[int8]*HandValue)
		t.Log("\n============ hand =============\n")
		for j := 0; j < 9; j++ {
			cs2, _ := a.GetCards(2)
			allcs := append(cs, cs2...)
			hand, err := GetMaxHandValueFromCard(allcs)
			if err != nil {
				t.Error(err)
				return
			}
			str := hand.String()
			str += hand.DebugCards(allcs)
			t.Log(str)
			hands[int8(j)] = hand
		}
		t.Log("\n============ max hand =============\n")
		maxHands := GetMaxHandValueFromTaggedHandValues(hands)
		for _, hand := range maxHands {
			t.Log(hand, hand.Value())
		}
		i--
		a.Reset()
	}
}

func TestCards(t *testing.T) {
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)

	a1, _ := NewCard(13, 0)
	a2, _ := NewCard(14, 2)
	b1, _ := NewCard(2, 1)
	b2, _ := NewCard(5, 0)

	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5, a1, a2})
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5, b1, b2})
	t.Log(r1.String())
	t.Log(r2.String())

	max := GetMaxHandValue(r1, r2)

	t.Log(max[0].String())

}

func TestHVHighCard(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(7, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVHighCard)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(8, 1)
	c3, _ = NewCard(4, 0)
	c4, _ = NewCard(5, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVHighCard)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVOnePair(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVOnePair)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(4, 1)
	c3, _ = NewCard(4, 0)
	c4, _ = NewCard(5, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVOnePair)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVTwoPair(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(12, 0)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVTwoPair)

	c1, _ = NewCard(11, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(13, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVTwoPair)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVThreeOfAKind(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(2, 0)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVThreeOfAKind)

	c1, _ = NewCard(11, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(11, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVThreeOfAKind)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVStraight(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraight)

	c1, _ = NewCard(10, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(12, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVStraight)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 1)
	c2, _ := NewCard(7, 1)
	c3, _ := NewCard(4, 1)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFlush)

	c1, _ = NewCard(3, 2)
	c2, _ = NewCard(7, 2)
	c3, _ = NewCard(4, 2)
	c4, _ = NewCard(5, 2)
	c5, _ = NewCard(14, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVFullHouse(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(3, 1)
	c2, _ := NewCard(3, 2)
	c3, _ := NewCard(3, 0)
	c4, _ := NewCard(10, 1)
	c5, _ := NewCard(10, 2)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFullHouse)

	c1, _ = NewCard(3, 1)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(3, 0)
	c4, _ = NewCard(14, 0)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFullHouse)
	assert.Equal(r2.Value() > r1.Value(), true)
}
func TestHVFourOfAKind(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(3, 1)
	c2, _ := NewCard(3, 2)
	c3, _ := NewCard(3, 0)
	c4, _ := NewCard(3, 3)
	c5, _ := NewCard(13, 2)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFourOfAKind)

	c1, _ = NewCard(3, 1)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(3, 0)
	c4, _ = NewCard(3, 3)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFourOfAKind)
	assert.Equal(r2.Value() > r1.Value(), true)
}
func TestHVStraightFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 1)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 1)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraightFlush)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(4, 2)
	c4, _ = NewCard(5, 2)
	c5, _ = NewCard(6, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVStraightFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVRoyalFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(9, 1)
	c2, _ := NewCard(10, 1)
	c3, _ := NewCard(11, 1)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(13, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraightFlush)

	c1, _ = NewCard(10, 2)
	c2, _ = NewCard(11, 2)
	c3, _ = NewCard(12, 2)
	c4, _ = NewCard(13, 2)
	c5, _ = NewCard(14, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVRoyalFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestCalcPots(t *testing.T) {
	h := &Holdem{}
	urs := make([]*Agent, 6)
	urs[0] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 1,
			handBet:    1000,
			status:     ActionDefAllIn,
		},
	}
	urs[1] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 2,
			handBet:    2000,
			status:     ActionDefAllIn,
		},
	}
	urs[2] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 3,
			handBet:    4000,
			status:     ActionDefAllIn,
		},
	}
	urs[3] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 4,
			handBet:    4000,
			status:     ActionDefAllIn,
		},
	}
	urs[4] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 5,
			handBet:    5000,
			status:     ActionDefAllIn,
		},
	}
	urs[5] = &Agent{
		gameInfo: &gameInfo{
			seatNumber: 6,
			handBet:    7000,
			status:     ActionDefAllIn,
		},
	}
	for i := 0; i < len(urs)-1; i++ {
		urs[i].nextAgent = urs[i+1]
		urs[i+1].prevAgent = urs[i]
	}
	urs[0].prevAgent = urs[len(urs)-1]
	urs[len(urs)-1].nextAgent = urs[0]
	h.button = urs[0]
	pots := h.calcPot(urs)
	for _, p := range pots {
		t.Log(p.Num, p.SeatNumber)
	}
}

func TestGetOuts(t *testing.T) {
	publicCards := make([]*Card, 4)
	publicCards[0], _ = NewCard(12, 0)
	publicCards[1], _ = NewCard(7, 1)
	publicCards[2], _ = NewCard(4, 1)
	publicCards[3], _ = NewCard(10, 0)

	mp := make(map[int8][]*Card)
	cards := make([]*Card, 2)
	cards[0], _ = NewCard(13, 0)
	cards[1], _ = NewCard(12, 3)
	mp[1] = cards

	cards = make([]*Card, 2)
	cards[0], _ = NewCard(5, 1)
	cards[1], _ = NewCard(6, 1)
	mp[2] = cards

	cards = make([]*Card, 2)
	cards[0], _ = NewCard(13, 1)
	cards[1], _ = NewCard(11, 1)
	mp[3] = cards

	cards = make([]*Card, 2)
	cards[0], _ = NewCard(7, 0)
	cards[1], _ = NewCard(2, 0)
	mp[4] = cards

	mp1, mp2 := GetAllOuts(publicCards, mp)

	potOuts := GetOuts(mp1, mp2, []*Pot{
		{
			SeatNumber: map[int8]bool{1: true, 2: true, 3: true, 4: true},
			Num:        1000,
		},
	})
	for leader, outs := range potOuts {
		for _, v := range outs.Outs {
			if v.Len > 0 {
				t.Log(v.Len)
				for seat, m := range v.Detail {
					for cd, vv := range m {
						t.Log(leader, seat, cd, vv)
					}
				}
			}
		}
	}
}

type TestAd struct {
	Num int
}

type TestAd2 struct {
	Ad *TestAd
}

func TestPointer(t *testing.T) {
	a := &TestAd{
		Num: 1,
	}
	b := &TestAd2{
		Ad: a,
	}
	c := &TestAd2{}
	c.Ad = b.Ad
	c.Ad.Num = 2

	t.Log(c.Ad, b.Ad)
}

func TestPonter2(t *testing.T) {
	A := &TestAd{Num: 1}
	B := &TestAd{Num: 2}
	R := &TestAd2{Ad: A}
	a := map[int]*TestAd2{1: R}
	b := map[int]*TestAd2{1: R}
	t.Log(b[1].Ad.Num)
	c := a[1]
	c.Ad = B
	a[1] = c
	t.Log(b[1].Ad.Num)
}

func TestGoRoutine(t *testing.T) {
	a := []*TestAd{
		{
			Num: 1,
		},
		{
			Num: 2,
		},

		{
			Num: 3,
		},
	}
	i := 0
	var u *TestAd
	u = a[i]
	for u != nil {
		t.Log(u.Num)
		i++
		var next *TestAd
		if i <= 2 {
			next = a[i]
		}
		this := u
		time.AfterFunc(2*time.Second, func() {
			if next != nil {
				t.Log(this.Num, next.Num)
			} else {
				t.Log(this.Num, "null")
			}
		})
		u = next
	}
	c := time.After(10 * time.Second)
	<-c
}

func TestChannel(t *testing.T) {
	th := make(chan bool)
	defer close(th)
	timer := time.NewTimer(5 * time.Second)
	go func() {
		time.Sleep(11 * time.Second)
		th <- true
	}()
	i := 0
	for {
	L:
		select {
		case <-th:
			t.Log("th")
			return
		case <-timer.C:
			i++
			if i < 2 {
				timer = time.NewTimer(5 * time.Second)
				break L
			}
			t.Log("time out", i)
			return
		}
	}
}
