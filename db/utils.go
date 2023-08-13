package db

import "time"

func mstime() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
