package main

import (
	"github.com/florisboom/go-notion-manga-tracker/cmd/crawler"
	"github.com/robfig/cron"
)

func main() {
	c := cron.New()
	c.AddFunc("0 30 * * * *", func() {
		crawler.Sync()
	})

	c.Start()
}
