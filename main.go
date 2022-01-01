package main

import (
	"github.com/florisboom/go-notion-manga-tracker/crawler"
	"github.com/robfig/cron/v3"
)

func main() {
	c := cron.New()
	c.AddFunc("0/1 * * * *", func() {
		crawler.Sync()
	})

	c.Start()
	select {}
}
