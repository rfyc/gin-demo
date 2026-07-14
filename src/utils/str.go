package utils

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sony/sonyflake"
)

var sf *sonyflake.Sonyflake
var timeLocal time.Time

func init() {
	timeLocal, _ = time.ParseInLocation("2006-01-02 15:04:05", "2024-06-11 15:00:00", time.Local)
	sf = sonyflake.NewSonyflake(sonyflake.Settings{
		StartTime: timeLocal,
	})
}

func StringToMd5(str string) string {
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}
func GetUUId() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// GlobalId get global id
func GlobalId() (uint64, error) {
	return sf.NextID()
}
