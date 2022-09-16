package main

import (
	"context"
	"fmt"
	xerr "github.com/goclub/error"
	"time"
)

func main() {
	ctx := context.Background()
	go func() {
		defer func() {
			// 防止 routine panic 导致程序退出
			r := recover()
			if r != nil {
				sentryClient.Error(xerr.New(fmt.Sprintf("%+v", r)))
			}
		}()
		refreshAccessTokenJob(ctx)
		for {
			time.Sleep(time.Minute)
			refreshAccessTokenJob(ctx)
		}
	}()
	go func() {
		defer func() {
			// 防止 routine panic 导致程序退出
			r := recover()
			if r != nil {
				sentryClient.Error(xerr.New(fmt.Sprintf("%+v", r)))
			}
		}()
		refreshAccessTokenJob(ctx)
		for {
			time.Sleep(time.Minute)
			refreshAccessTokenJob(ctx)
		}
	}()
	err := httpListen() // indivisible begin
	if err != nil {     // indivisible end
		sentryClient.Error(err)
	}
}
