package main

import (
	"github.com/florisboom/go-notion-manga-tracker/crawler"
	"github.com/robfig/cron"
)

func main() {
	c := cron.New()
	c.AddFunc("59 0/1 * * *", func() {
		crawler.Sync()
	})

	c.Start()
	select {}
}
