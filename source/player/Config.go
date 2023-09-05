package player

import (
	"time"

	"meowyplayer.com/source/utility"
)

type Config struct {
	utility.Subject[*Config]
	Date   time.Time `json:"date"`
	Albums []Album   `json:"albums"`
}

func (c *Config) NotifyAll() {
	c.Subject.NotifyAll(c)
}