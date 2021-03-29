package holdem

import (
	"io"
	"os"
	"testing"

	"github.com/whatisfaker/holdem"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func parseLevelString(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	}
	return zapcore.InfoLevel
}

func newLogger(level string, wr io.Writer) *zap.Logger {
	logLevel := zap.NewAtomicLevel()
	logLevel.SetLevel(parseLevelString(level))
	w := zapcore.AddSync(wr)
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		w,
		logLevel,
	)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

func TestNewHoldem(t *testing.T) {
	l := newLogger("info", os.Stdout)
	defer l.Sync()
	l2 := zap.NewNop()
	defer l.Sync()
	h := holdem.NewHoldem(6, 100, 0, l.With(zap.String("game", "holdem")))
	a1 := newTestAgent(l2)
	a2 := newTestAgent(l.With(zap.String("agent", "a2-5")))
	a3 := newTestAgent(l2)
	h.Join(a1, a2, a3)
	a1.BringIn(1000)
	a2.BringIn(1000)
	a3.BringIn(1000)
	h.Seated(0, a1)
	h.Seated(5, a2)
	h.Seated(2, a3)
	h.StartHand()
	t.Log("fin")
	// h.StartHand()
	// t.Log("fin2", h.buttonSeat)
	// h.StartHand()
	// t.Log(h.buttonSeat)
}

func TestCaculatePots(t *testing.T) {
	l, _ := zap.NewDevelopment()
	defer l.Sync()
	h := holdem.NewHoldem(6, 100, 0, l.With(zap.String("game", "holdem")))
	A := newTestAgent(l)
	U := newTestAgent(l)
	B := newTestAgent(l)
	C := newTestAgent(l)
	D := newTestAgent(l)
	h.Join(A, U, B, C, D)
	A.BringIn(10000)
	h.Seated(0, A)
	U.BringIn(10000)
	h.Seated(1, U)
	B.BringIn(10000)
	h.Seated(2, B)
	C.BringIn(10000)
	h.Seated(3, C)
	D.BringIn(10000)
	h.Seated(4, D)

	// h.buttonPosition()
	// h.deal(2)
	// h.dealPublicCards(5)

	// A.gameInfo.handBet = 2941 //3990
	// A.gameInfo.status = ActionDefCall
	// U.gameInfo.handBet = 1450
	// U.gameInfo.status = ActionDefAllIn
	// B.gameInfo.handBet = 2941
	// B.gameInfo.status = ActionDefAllIn
	// C.gameInfo.handBet = 947
	// C.gameInfo.status = ActionDefAllIn
	// D.gameInfo.handBet = 21
	// D.gameInfo.status = ActionDefFold

	// m := []*Agent{
	// 	A, U, B, C,
	// }
	// h.complexWin(m)
	// // pots := h.calcPot(m)
	// // for _, v := range pots {
	// // 	t.Log("pot", v.Num, v.SeatNumber)
	// // }
	// // result, _, _ := h.calcWin(m, pots)
	// // for _, v := range result {
	// // 	t.Log("result", v.SeatNumber, v.Num)
	// // }

}
