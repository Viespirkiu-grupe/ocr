package model

import (
	"strconv"
	"time"
)

type Task struct {
	ID      int       `json:"id"`
	Uri     string    `json:"uri"`
	Expires time.Time `json:"expires"`
}

func (t Task) IDString() string {
	return strconv.Itoa(t.ID)
}
