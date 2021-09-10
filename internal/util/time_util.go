package util

import (
	"strconv"
	"time"
)

type TimeUtil struct {
}

func (u TimeUtil) GetTimeStr() string {
	return time.Now().Format("2006-01-02 15:04:05.999999999") + " " + strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
}
