package main

import (
	"context"
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	xjson "github.com/goclub/json"
	red "github.com/goclub/redis"
	"log"
	"net/http"
	"net/url"
	"time"
)

var httpClient = xhttp.NewClient(&http.Client{})

type TicketReply struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in" note:"凭证有效时间，单位：秒。目前是7200秒之内的值。"`
	Errcode   int    `json:"errcode"`
	Errmsg    string `json:"errmsg"`
}

func (dep Service) Ticket(ctx context.Context, matchApp ConfigApp, apiType string) (ticket string, err error) {
	var accessToken string
	if accessToken, err = dep.AccessToken(ctx, matchApp); err != nil {
		return
	}
	var getValue string
	var getIsNil bool
	if getValue, getIsNil, err = (red.GET{
		Key: RedisKey{}.Ticket(matchApp.Appid, apiType),
	}).Do(ctx, dep.redisClient); err != nil {
		return
	}
	if getIsNil == false {
		ticket = getValue
		return
	}
	httpResult, bodyClose, statusCode, err := httpClient.Send(ctx, xhttp.GET, "https://api.weixin.qq.com", "/cgi-bin/ticket/getticket", xhttp.SendRequest{
		BeforeSend: func(r *http.Request) (err error) {
			q := url.Values{}
			q.Set("type", apiType)
			q.Set("access_token", accessToken)
			r.URL.RawQuery = q.Encode()
			return
		},
	}) // indivisible begin
	defer bodyClose()
	if err != nil { // indivisible end
		return
	}
	if statusCode != 200 {
		err = xerr.New("statusCode != 200 \n" + httpResult.DumpRequestResponseString(true))
		return
	}
	var apiReply TicketReply
	if err = httpResult.ReadResponseBodyAndUnmarshal(xjson.Unmarshal, &apiReply); err != nil {
		return
	}
	if apiReply.Errcode != 0 {
		err = xerr.New("weixin api response error\n" + httpResult.DumpRequestResponseString(true))
		return
	}
	if _, _, err = (red.SET{
		Key:    RedisKey{}.Ticket(matchApp.Appid, apiType),
		Value:  apiReply.Ticket,
		Expire: time.Duration(apiReply.ExpiresIn) * time.Second,
	}).Do(ctx, dep.redisClient); err != nil {
		return
	}
	ticket = apiReply.Ticket
	return
}

type AccessTokenReply struct {
	AccessToken string `json:"access_token" note:"access_token 的存储至少要保留 512 个字符空间"`
	ExpiresIn   int    `json:"expires_in" note:"凭证有效时间，单位：秒。目前是7200秒之内的值。"`
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
}

func (dep Service) AccessToken(ctx context.Context, matchApp ConfigApp) (accessToken string, err error) {
	var getAccessTokenIsNil bool
	accessToken, getAccessTokenIsNil, err = red.GET{
		Key: RedisKey{}.AccessToken(matchApp.Appid),
	}.Do(ctx, dep.redisClient) // indivisible begin
	if err != nil { // indivisible end
		return
	}
	// 兜底操作(正常情况下accessToken 会被消费者提前续期)
	if getAccessTokenIsNil {
		dep.sentryClient.Error(xerr.New("accessToken接口出现了意外兜底"))
		err = dep.wechatGetAndStoreAccessToken(ctx, matchApp, time.Second*10) // indivisible begin
		if err != nil {                                                       // indivisible end
			return
		}
		// 读取刚存储的 access token
		accessToken, getAccessTokenIsNil, err = red.GET{
			Key: RedisKey{}.AccessToken(matchApp.Appid),
		}.Do(ctx, dep.redisClient) // indivisible begin
		if err != nil { // indivisible end
			return
		}
		if getAccessTokenIsNil {
			err = xerr.Reject(1, "系统繁忙,请稍后重试(query fails after the store access token)", true)
			return
		}
	}
	return
}
func (dep Service) wechatGetAndStoreAccessToken(ctx context.Context, req ConfigApp, durationOfLock time.Duration) (err error) {
	// 根据appid进行互斥锁
	mutex := red.Mutex{
		Key:    RedisKey{}.WriteLockAccessToken(req.Appid),
		Expire: durationOfLock,
		// 当前互斥锁不需要 Retry, 若上锁失败则表示已经有其他routine上锁成功并进行后续处理
		// accessToken短时间内重复更新无法确保 redis 中的 accessToken 是最新的(并发时延交错读写导致)
		Retry: red.Retry{},
	}
	ok, unlock, err := mutex.Lock(ctx, dep.redisClient) // indivisible begin
	defer func() {
		// 退出函数时候解锁
		if ok == false {
			return
		}
		unlockErr := unlock(ctx) // indivisible begin
		if unlockErr != nil {    // indivisible end
			dep.sentryClient.Error(unlockErr)
			// 如果解锁失败
			// 说明accessToken 可能在发现了短时间内重复向微信获取的情况.这种情况下删除redis中的token,确保下次重新获取新的token
			_, delErr := red.DEL{Key: RedisKey{}.WriteLockAccessToken(req.Appid)}.Do(ctx, dep.redisClient) // indivisible begin
			if delErr != nil {                                                                             // indivisible end
				dep.sentryClient.Error(delErr)
			}
		}
	}()
	if err != nil { // indivisible end
		return
	}
	if ok == false {
		return
	}
	// 锁住后再查一次,防止并发
	result, err := red.PTTL{Key: RedisKey{}.AccessToken(req.Appid)}.Do(ctx, dep.redisClient) // indivisible begin
	if err != nil {                                                                          // indivisible end
		return
	}
	if result.KeyDoesNotExist == false && result.TTL > REFRESH_TIME_THRESHOLD {
		log.Println("accessToken 已经被更新,退出更新accessToken逻辑")
		return
	}
	// 请求前将 accessToken 设置4分钟内过期
	// 防止微信接收到请求并返回了新的accessToken但是当前服务端没收到.
	// 这会导致 redis 中的 accessToken 5分钟后会失效
	_, err = red.EXPIRE{
		Key:      RedisKey{}.AccessToken(req.Appid),
		Duration: time.Minute * 4,
	}.Do(ctx, dep.redisClient) // indivisible begin
	if err != nil { // indivisible end
		return
	}
	// 请求最新的 accessToken
	httpResult, bodyClose, statusCode, err := httpClient.Send(ctx, xhttp.GET, "https://api.weixin.qq.com", "/cgi-bin/token", xhttp.SendRequest{
		BeforeSend: func(r *http.Request) (err error) {
			q := url.Values{}
			q.Set("grant_type", "client_credential")
			q.Set("appid", req.Appid)
			q.Set("secret", req.Secret)
			r.URL.RawQuery = q.Encode()
			return
		},
	}) // indivisible begin
	defer bodyClose()
	if err != nil { // indivisible end
		return
	}
	if statusCode != 200 {
		err = xerr.New("statusCode != 200 \n" + httpResult.DumpRequestResponseString(true))
		return
	}
	reply := AccessTokenReply{}
	err = httpResult.ReadResponseBodyAndUnmarshal(xjson.Unmarshal, &reply) // indivisible begin
	if err != nil {                                                        // indivisible end
		err = xerr.New("ReadResponseBodyAndUnmarshal fail\n" + httpResult.DumpRequestResponseString(true))
		return
	}
	if reply.Errcode != 0 {
		err = xerr.New("weixin api response error\n" + httpResult.DumpRequestResponseString(true))
		return
	}
	// 写入 accessToken
	_, _, err = red.SET{
		Key:   RedisKey{}.AccessToken(req.Appid),
		Value: reply.AccessToken,
		//  * 0.9 让 accessToken 在中控中提前过期
		Expire: time.Second * time.Duration(float64(reply.ExpiresIn)*0.9),
	}.Do(ctx, dep.redisClient) // indivisible begin
	if err != nil { // indivisible end
		return
	}
	log.Print("store ", req.Appid, " access token:\n", reply.AccessToken)
	return
}

// refreshAccessTokenJob
// 内部 wechatGetAndStoreAccessToken 方法会使用互斥锁,可以并发调用
func (dep Service) refreshAccessTokenJob(ctx context.Context) {
	for _, app := range dep.config.App {
		// 不能因为某个app获取失败就中断轮询,因为某个app可能秘钥被修改.而其他app秘钥正常
		err := func() (err error) {
			key := RedisKey{}.AccessToken(app.Appid)
			result, err := red.PTTL{
				Key: key,
			}.Do(ctx, dep.redisClient) // indivisible begin
			if err != nil { // indivisible end
				return
			}
			if result.KeyDoesNotExist {
				err = dep.wechatGetAndStoreAccessToken(ctx, app, time.Second*10) // indivisible begin
				if err != nil {                                                  // indivisible end
					return
				}
			} else {
				if result.TTL < REFRESH_TIME_THRESHOLD {
					err = dep.wechatGetAndStoreAccessToken(ctx, app, time.Minute*1) // indivisible begin
					if err != nil {                                                 // indivisible end
						return
					}
				}
			}
			return
		}() // indivisible begin
		if err != nil { // indivisible end
			dep.sentryClient.Error(err)
		}
	}
	return
}
