package vivopush

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
)

var authToken *AuthToken = new(AuthToken)

type VivoClient struct {
	AppId      string
	AppKey     string
	AppSecret  string
	tokenCache TokenCache
}

type VivoTokenPar struct {
	AppId     string `json:"appId"`
	AppKey    string `json:"appKey"`
	Timestamp int64  `json:"timestamp"`
	Sign      string `json:"sign"`
}

type AuthToken struct {
	token      string
	valid_time int64
}

type VivoPush struct {
	host       string
	vc         *VivoClient
	tokenCache TokenCache
}

func NewClient(appId, appKey, appSecret string, tokenCache TokenCache) (*VivoPush, error) {
	vc := &VivoClient{
		appId,
		appKey,
		appSecret,
		tokenCache,
	}
	return &VivoPush{
		host: ProductionHost,
		vc:   vc,
	}, nil
}

//----------------------------------------Token----------------------------------------//
//获取token
func (vc *VivoClient) GetToken() (string, error) {
	now := time.Now().UnixNano() / 1e6
	if authToken != nil {
		if authToken.valid_time > now {
			return authToken.token, nil
		}
	}
	// 从缓存中获取
	if vc.tokenCache != nil {
		ti, err := vc.tokenCache.TokenCache(vc.AppId, vc.AppKey, vc.AppSecret)
		if err != nil {
			return "", err
		}
		if ti != nil {
			authToken.token = ti.Token
			authToken.valid_time = ti.TokenValidTime
		}
	} else {
		token, err := GetTokenByRequest(vc.AppId, vc.AppKey, vc.AppSecret)
		if err != nil {
			return "", err
		}
		if token != "" {
			authToken.token = token
			authToken.valid_time = now + 3600000 //1小时有效
		}
	}
	return authToken.token, nil
}

func GetTokenByRequest(appId, appKey, appSecret string) (string, error) {
	now := time.Now().UnixNano() / 1e6
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(appId + appKey + strconv.FormatInt(now, 10) + appSecret))
	sign := hex.EncodeToString(md5Ctx.Sum(nil))

	formData, err := json.Marshal(&VivoTokenPar{
		AppId:     appId,
		AppKey:    appKey,
		Timestamp: now,
		Sign:      sign,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", ProductionHost+AuthURL, bytes.NewReader(formData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	result, err := handleResponse(resp)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("network error")
	}
	js, err := simplejson.NewJson(result)
	if err != nil {
		return "", err
	}
	token, err := js.Get("authToken").String()
	if err != nil {
		return "", err
	}
	return token, nil
}

//----------------------------------------Sender----------------------------------------//
// 根据regID，发送消息到指定设备上
func (v *VivoPush) Send(msg *Message, regID string) (*SendResult, error) {
	params := v.assembleSendParams(msg, regID)
	res, err := v.doPost(v.host+SendURL, params)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// 保存群推消息公共体接口
func (v *VivoPush) SaveListPayload(msg *MessagePayload) (*SendResult, error) {
	res, err := v.doPost(v.host+SaveListPayloadURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// 群推
func (v *VivoPush) SendList(msg *MessagePayload, regIds []string) (*SendResult, error) {
	if len(regIds) < 2 || len(regIds) > 1000 {
		return nil, errors.New("regIds个数必须大于等于2,小于等于 1000")
	}
	res, err := v.SaveListPayload(msg)
	if err != nil {
		return nil, err
	}
	if res.Result != 0 {
		return nil, errors.New(fmt.Sprintf("save list payload fail; res=%v", res))
	}
	bs, err := json.Marshal(NewListMessage(regIds, res.RequestId))
	if err != nil {
		return nil, err
	}
	//推送
	res2, err := v.doPost(v.host+PushToListURL, bs)
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res2, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// 全量推送
func (v *VivoPush) SendAll(msg *MessagePayload) (*SendResult, error) {
	res2, err := v.doPost(v.host+PushToAllURL, msg.JSON())
	if err != nil {
		return nil, err
	}
	var result SendResult
	err = json.Unmarshal(res2, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//----------------------------------------Tracer----------------------------------------//
// 获取指定消息的状态。
func (v *VivoPush) GetMessageStatusByJobKey(jobKey string) (*BatchStatusResult, error) {
	params := v.assembleStatusByJobKeyParams(jobKey)
	res, err := v.doGet(v.host+MessagesStatusURL, params)
	if err != nil {
		return nil, err
	}
	var result BatchStatusResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (v *VivoPush) assembleSendParams(msg *Message, regID string) []byte {
	msg.RegId = regID
	jsondata := msg.JSON()
	return jsondata
}

func (v *VivoPush) assembleStatusByJobKeyParams(jobKey string) string {
	form := url.Values{}
	form.Add("taskIds", jobKey)
	return "?" + form.Encode()
}

func handleResponse(response *http.Response) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (v *VivoPush) doPost(url string, formData []byte) ([]byte, error) {
	token, e := v.vc.GetToken()
	if e != nil {
		return nil, e
	}
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.NewRequest("POST", url, bytes.NewReader(formData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", token)
	client := &http.Client{}
	tryTime := 0
tryAgain:
	resp, err = client.Do(req)
	if err != nil {
		tryTime += 1
		if tryTime < PostRetryTimes {
			goto tryAgain
		}
		return nil, err
	}
	result, err = handleResponse(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("network error")
	}
	return result, nil
}

func (v *VivoPush) doGet(url string, params string) ([]byte, error) {
	token, e := v.vc.GetToken()
	if e != nil {
		return nil, e
	}
	var result []byte
	var req *http.Request
	var resp *http.Response
	var err error
	req, err = http.NewRequest("GET", url+params, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authToken", token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	result, err = handleResponse(resp)
	if err != nil {
		return nil, err
	}
	return result, nil
}
