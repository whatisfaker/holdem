package holdem

type Recorder interface {
	GameStart(map[string]interface{})
	HandBegin(*HoldemState)
	Ante(meta map[string]interface{}, seat int8, chip uint, num uint)
	Action(meta map[string]interface{}, round Round, seat int8, chip uint, action ActionDef, num uint)
	InsureResult(meta map[string]interface{}, round Round, seat int8, bet uint, win float64)
	HandEnd(meta map[string]interface{}, r []*Result)
	GameEnd(meta map[string]interface{})
}

type nopRecorder struct {
}

func newNopRecorder() Recorder {
	return &nopRecorder{}
}

var _ Recorder = (*nopRecorder)(nil)

func (c *nopRecorder) GameStart(map[string]interface{}) {}

func (c *nopRecorder) GameEnd(map[string]interface{}) {}

func (c *nopRecorder) HandBegin(*HoldemState) {}

func (c *nopRecorder) Ante(meta map[string]interface{}, seat int8, chip uint, num uint) {}

func (c *nopRecorder) Action(meta map[string]interface{}, round Round, seat int8, chip uint, action ActionDef, num uint) {
}

func (c *nopRecorder) InsureResult(meta map[string]interface{}, round Round, seat int8, bet uint, win float64) {
}

func (c *nopRecorder) HandEnd(meta map[string]interface{}, r []*Result) {}
