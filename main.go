package main

import (
	"context"
	"fmt"
	xerr "github.com/goclub/error"
	"time"
)

func main() {
	ctx := context.Background()
	service, err := NewService() // indivisible begin
	if err != nil {              // indivisible end
		panic(err)
	}

	go func() {
		defer func() {
			// 防止 routine panic 导致程序退出
			r := recover()
			if r != nil {
				service.sentryClient.Error(xerr.New(fmt.Sprintf("%+v", r)))
			}
		}()
		service.refreshAccessTokenJob(ctx)
		for {
			time.Sleep(time.Minute)
			service.refreshAccessTokenJob(ctx)
		}
	}()
	err = service.httpListen() // indivisible begin
	if err != nil {            // indivisible end
		service.sentryClient.Error(err)
	}
}
