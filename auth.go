package main

import (
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	"strings"
)

func (dep Service) AuthAndMatch(c *xhttp.Context) (matchApp ConfigApp, err error) {
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
	for _, app := range dep.config.App {
		if req.Appid == app.Appid {
			matchAppid = true
			matchApp = app
			break
		}
	}
	if matchAppid == false {
		err = xerr.Reject(1, "appid 不存在("+req.Appid+")", true)
		return
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
		err = xerr.Reject(1, "sk 错误", true)
		return
	}
	return
}
