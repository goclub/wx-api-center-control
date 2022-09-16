package main

import (
	sentry "github.com/getsentry/sentry-go"
	"log"
)

var sentryClient = SentryClient{}

type SentryClient struct {
	hub *sentry.Hub
}

func (c SentryClient) Error(err error) {
	log.Printf("%+v", err)
	if c.hub != nil {
		c.hub.CaptureException(err)
	}
}
