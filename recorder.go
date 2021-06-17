package holdem

type Recorder interface {
	Begin(*HoldemState)
	Ante(seat int8, chip int, num int)
	Action(round Round, seat int8, chip int, action ActionDef, num int)
	InsureResult(round Round, seat int8, bet int, win float64)
	End([]*Result)
}

type nopRecorder struct {
}

func newNopRecorder() Recorder {
	return &nopRecorder{}
}

var _ Recorder = (*nopRecorder)(nil)

func (c *nopRecorder) Begin(*HoldemState) {}

func (c *nopRecorder) Ante(seat int8, chip int, num int) {}

func (c *nopRecorder) Action(round Round, seat int8, chip int, action ActionDef, num int) {}

func (c *nopRecorder) InsureResult(round Round, seat int8, bet int, win float64) {}

func (c *nopRecorder) End([]*Result) {}
