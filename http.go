package main

import (
	xerr "github.com/goclub/error"
	xhttp "github.com/goclub/http"
	"net/http"
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
	router.HandleFunc(xhttp.Route{xhttp.POST, "/wx-api-center-control/cgi-bin/ticket/getticket"}, func(c *xhttp.Context) (err error) {
		ctx := c.Request.Context()
		var matchApp ConfigApp
		if matchApp, err = dep.AuthAndMatch(c); err != nil {
			return
		}
		apiType := c.Request.URL.Query().Get("type")
		if apiType == "" {
			return xerr.Reject(1, "query 中的 type 不能为空", true)
		}
		var ticket string
		if ticket, err = dep.Ticket(ctx, matchApp, apiType); err != nil {
			return
		}
		return c.WriteJSON(struct {
			xerr.Resp
			Ticket string `json:"ticket"`
		}{
			Ticket: ticket,
		})
	})
	router.HandleFunc(xhttp.Route{xhttp.POST, "/wx-api-center-control/cgi-bin/token"}, func(c *xhttp.Context) (err error) {
		ctx := c.Request.Context()
		var matchApp ConfigApp
		if matchApp, err = dep.AuthAndMatch(c); err != nil {
			return
		}
		var accessToken string
		if accessToken, err = dep.AccessToken(ctx, matchApp); err != nil {
			return
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
