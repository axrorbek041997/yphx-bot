package utils

func UserCacheKey(tgUserID int64) string {
	return "user:" + string(rune(tgUserID))
}
