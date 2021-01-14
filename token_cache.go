package vivopush

var tokenCache TokenCache

type TokenCache interface {
	// 获取token信息，若为空则获取 token 加入
	TokenCache(appId, appKey, appSecret string) (*TokenInfo, error)
}

func InitTokenCache(t TokenCache) {
	tokenCache = t
}

type TokenInfo struct {
	Token string `json:"token"`
	TokenValidTime int64 `json:"token_valid_time"`
	KeyExpire int64 `json:"key_expire"`
}