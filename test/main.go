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
	s := example.NewServer(time.Now(), time.Now().Add(1*time.Minute), 2, sl)
	for i := 0; i < 12; i++ {
		//l := example.NewLogger("debug", os.Stdout)
		l := zap.NewNop()
		// // if i == 0 {
		// l := example.NewLogger("error", os.Stdout)
		// //}
		j := i
		a := example.NewRobot(j, l)
		if i < 2 {
			s.Connect(a)
		} else {
			s.Connect2(a)
		}
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
