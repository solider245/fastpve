package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func ToString(value interface{}) string {
	return fmt.Sprintf("%+v", value)
}

func CleanString(s string) string {
	// 匹配中文、英文、数字、空格
	reg := regexp.MustCompile(`[^\p{Han}a-zA-Z0-9-\s]`)
	s2 := reg.ReplaceAllString(s, "")
	s2 = strings.Replace(s2, " ", "-", -1)
	s2 = strings.Replace(s2, "_", "-", -1)
	return strings.ToLower(s2)
}

func ResetTimer(t *time.Timer, dur time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(dur)
}
