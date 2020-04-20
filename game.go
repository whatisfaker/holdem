package holdem

const (
	SeatStatusEmpty int8 = iota + 1
	SeatStatusTaken
	SeatStatusSeated
)

const (
	StepNotStarted int8 = iota - 1
	StepPreFlopRound
	StepFlopRound
	StepTurnRound
	StepRiverRound
)

type Game struct {
	poker       *Poker
	burnCards   []*Card
	publicCards []*Card
	handCards   map[int8][2]*Card
	playerCount int8
	players     map[int8]*Player
	seated      map[int8]int8
	step        int8
}

func NewGame(count int8) *Game {
	g := &Game{
		poker:       NewPoker(),
		burnCards:   make([]*Card, 0),
		publicCards: make([]*Card, 0),
		handCards:   make(map[int8][2]*Card),
		playerCount: count,
		players:     make(map[int8]*Player),
		seated:      make(map[int8]int8),
	}
	var i int8
	for i = 0; i < count; i++ {
		g.seated[i] = SeatStatusEmpty
	}
	return g
}

func (c *Game) BurnCard() error {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return err
	}
	c.burnCards = append(c.burnCards, cards...)
	return nil
}

func (c *Game) HandCard() ([]*Card, error) {
	cards, err := c.poker.GetCards(2)
	if err != nil {
		return nil, err
	}
	return cards, nil
}

func (c *Game) Flop() ([]*Card, error) {
	cards, err := c.poker.GetCards(3)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards, nil
}

func (c *Game) Turn() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *Game) River() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *Game) Next() {
	c.step++
	switch c.step {
	case StepPreFlopRound:

	}
}
