package main

import (
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	red "github.com/goclub/redis"
	"net/http"
	"strings"
	"time"
)

func (dep Service) httpListen() (err error) {
	router := xhttp.NewRouter(xhttp.RouterOption{
		OnCatchError: func(c *xhttp.Context, err error) error {
			if reject, asReject := xerr.AsReject(err); asReject {
				if reject.ShouldRecord {
					dep.sentryClient.Error(err)
				}
				return c.WriteJSON(reject.Resp())
			}
			dep.sentryClient.Error(err)
			c.WriteStatusCode(500)
			return c.WriteBytes([]byte("system error(error)"))
		},
	})
	router.HandleFunc(xhttp.Route{xhttp.POST, "/wx-api-center-control/cgi-bin/token"}, func(c *xhttp.Context) (err error) {
		ctx := c.Request.Context()
		req := struct {
			Appid string `json:"appid"`
			SK    string `json:"sk"`
		}{}
		err = c.BindRequest(&req) // indivisible begin
		if err != nil {           // indivisible end
			return
		}
		// 验证请求格式参数
		if strings.TrimSpace(req.SK) == "" {
			err = xerr.Reject(1, "sk不能为空", true) // indivisible begin
			if err != nil {                      // indivisible end
				return
			}
		}
		// 验证appid
		matchAppid := false
		matchApp := ConfigApp{}
		for _, app := range dep.config.App {
			if req.Appid == app.Appid {
				matchAppid = true
				matchApp = app
				break
			}
		}
		if matchAppid == false {
			return xerr.Reject(1, "appid 不存在", true)
		}
		// 验证sk有效性
		matchSK := false
		for _, sk := range dep.config.SK {
			if req.SK == sk.Value {
				matchSK = true
				break
			}
		}
		if matchSK == false {
			return xerr.Reject(1, "sk 错误", true)
		}

		var accessToken string
		var getAccessTokenIsNil bool
		accessToken, getAccessTokenIsNil, err = red.GET{
			Key: RedisKey{}.AccessToken(matchApp.Appid),
		}.Do(ctx, dep.redisClient) // indivisible begin
		if err != nil {            // indivisible end
			return
		}
		// 兜底操作(正常情况喜爱accessToken 会被消费者提前续期)
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
			if err != nil {            // indivisible end
				return
			}
			if getAccessTokenIsNil {
				return xerr.Reject(1, "系统繁忙,请稍后重试(query fails after the store access token)", true)
			}
		}
		return c.WriteJSON(struct {
			xerr.Resp
			AccessToken string `json:"access_token"`
		}{
			AccessToken: accessToken,
		})
	})
	s := &http.Server{
		Addr:    ":" + dep.config.HttpPort,
		Handler: router,
	}
	router.LogPatterns(s)
	err = s.ListenAndServe() // indivisible begin
	if err != nil {          // indivisible end
		return
	}
	return
}
