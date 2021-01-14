package vivopush

var tokenCache TokenCache

type TokenCache interface {
	// 设置成功后 删除全局锁
	Set(t *TokenInfo) error
	// 获取token信息，若为空则返回 nil。若为空则需要重新获取，需要同步保证一致
	Get() (*TokenInfo, error)
}

func InitTokenCache(t TokenCache) {
	tokenCache = t
}

type TokenInfo struct {
	Token string `json:"token"`
	TokenValidTime int64 `json:"token_valid_time"`
	KeyExpire int64 `json:"key_expire"`
}