package main

import red "github.com/goclub/redis"

var redisClient = red.GoRedisV8{}

type RedisKey struct {
}

func (RedisKey) AccessToken(appid string) string {
	return "wx_api_center_control:access_token:appid:" + appid
}
func (RedisKey) WriteLockAccessToken(appid string) string {
	return "wx_api_center_control:write_lock:access_token:appid:" + appid
}
