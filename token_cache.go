package vivopush

type TokenCache interface {
	// 获取token信息，若为空则获取 token 加入
	TokenCache(appId, appKey, appSecret string) (*TokenInfo, error)
}

type TokenInfo struct {
	Token          string `json:"token"`
	TokenValidTime int64  `json:"token_valid_time"`
	KeyExpire      int64  `json:"key_expire"`
}
