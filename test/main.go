package main

import (
	"os"
	"time"

	"github.com/whatisfaker/holdem/test/example"
	"go.uber.org/zap"
)

func main() {
	sl := example.NewLogger("debug", os.Stdout)
	//sl := zap.NewNop()
	s := example.NewServer(time.Now(), time.Now().Add(1*time.Minute), 1, sl)
	for i := 0; i < 9; i++ {
		l := zap.NewNop()
		// if i == 0 {
		// 	l = example.NewLogger("debug", os.Stdout)
		// }
		a := example.NewRobot(l)
		s.Connect(a)
		time.AfterFunc(2*time.Second, a.Start)
	}
	for {
		if s.IsComplete() {
			ch := time.After(4 * time.Second)
			<-ch
			break
		}
	}
}
