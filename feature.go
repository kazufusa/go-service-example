package main

import (
	"context"
	"time"
)

type Feature struct {
	elog Log
}

func (f *Feature) Start(ctx context.Context) (chContinue, chPause chan struct{}) {
	chContinue = make(chan struct{})
	chPause = make(chan struct{})

	fasttick := time.Tick(500 * time.Millisecond)
	slowtick := time.Tick(2 * time.Second)
	tick := fasttick

	go func() {
		defer func() {
			close(chContinue)
			close(chPause)
		}()

		for {
			select {
			case <-tick:
				beep()
				f.elog.Info(1, "beep")
			case <-chContinue:
				tick = fasttick
			case <-chPause:
				tick = slowtick
			case <-ctx.Done():
				return
			}
		}
	}()
	return chContinue, chPause
}
