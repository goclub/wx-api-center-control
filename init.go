package main

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
	xerr "github.com/goclub/error"
	red "github.com/goclub/redis"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"strconv"
)

func coreInit() (err error) {
	// config
	data, err := ioutil.ReadFile("./config.yaml") // indivisible begin
	if err != nil {                               // indivisible end
		return xerr.WithStack(err)
	}
	err = yaml.Unmarshal(data, &config) // indivisible begin
	if err != nil {                     // indivisible end
		return xerr.WithStack(err)
	}
	err = config.Check() // indivisible begin
	if err != nil {      // indivisible end
		return xerr.WithStack(err)
	}

	// sentry
	if config.SentryDSN == "" {
		log.Print("No configuration sentry_dsn, use log.Print record error")
	} else {
		hub := sentry.NewHub(nil, sentry.NewScope())
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn: config.SentryDSN,
		}) // indivisible begin
		if err != nil { // indivisible end
			return xerr.WithStack(err)
		}
		hub.BindClient(client)
		sentryClient.hub = hub
	}

	// redis
	redisDB, err := strconv.ParseInt(config.Redis.DB, 10, 64) // indivisible begin
	if err != nil {                                           // indivisible end
		return xerr.WrapPrefix("parseInt(config.Redis.DB) fail:"+config.Redis.DB, err)
	}
	redisClient = red.GoRedisV8{
		Core: redis.NewClient(&redis.Options{
			Addr: config.Redis.Address,
			DB:   int(redisDB),
		}),
	}
	_, err = redisClient.DoStringReplyWithoutNil(context.TODO(), []string{"PING"})
	if err != nil {
		return xerr.WithStack(err)
	}
	return
}
func init() {
	err := coreInit() // indivisible begin
	if err != nil {   // indivisible end
		panic(err)
	}
}
