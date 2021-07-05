package holdem

type Recorder interface {
	GameStart(*HoldemBase)
	HandBegin(*HoldemState)
	Ante(base *HoldemBase, seat int8, id string, chip uint, num uint)
	Action(base *HoldemBase, round Round, seat int8, id string, chip uint, action ActionDef, num uint)
	InsureResult(base *HoldemBase, round Round, seat int8, id string, bet uint, win float64)
	HandEnd(state *HoldemState, r []*Result)
	GameEnd(base *HoldemBase)
}

type nopRecorder struct {
}

func newNopRecorder() Recorder {
	return &nopRecorder{}
}

var _ Recorder = (*nopRecorder)(nil)

func (c *nopRecorder) GameStart(*HoldemBase) {}

func (c *nopRecorder) GameEnd(*HoldemBase) {}

func (c *nopRecorder) HandBegin(*HoldemState) {}

func (c *nopRecorder) Ante(meta *HoldemBase, seat int8, id string, chip uint, num uint) {
}

func (c *nopRecorder) Action(meta *HoldemBase, round Round, seat int8, id string, chip uint, action ActionDef, num uint) {
}

func (c *nopRecorder) InsureResult(meta *HoldemBase, round Round, seat int8, id string, bet uint, win float64) {
}

func (c *nopRecorder) HandEnd(state *HoldemState, r []*Result) {}
