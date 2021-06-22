package holdem

type Recorder interface {
	Begin(*HoldemState)
	Ante(seat int8, chip uint, num uint)
	Action(round Round, seat int8, chip uint, action ActionDef, num uint)
	InsureResult(round Round, seat int8, bet uint, win float64)
	End([]*Result)
}

type nopRecorder struct {
}

func newNopRecorder() Recorder {
	return &nopRecorder{}
}

var _ Recorder = (*nopRecorder)(nil)

func (c *nopRecorder) Begin(*HoldemState) {}

func (c *nopRecorder) Ante(seat int8, chip uint, num uint) {}

func (c *nopRecorder) Action(round Round, seat int8, chip uint, action ActionDef, num uint) {}

func (c *nopRecorder) InsureResult(round Round, seat int8, bet uint, win float64) {}

func (c *nopRecorder) End([]*Result) {}
