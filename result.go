package vivopush

type User struct {
	Status int `json:"status"`
	UserId string `json:"userid"`
}

type ResultItem struct {
	Result int    `json:"result"`
	Desc   string `json:"desc"`
	InvalidUsers []User `json:"invalidUsers"`
}

type SendResult struct {
	ResultItem
	RequestId string `json:"requestId"`
}

type BatchStatusResult struct {
	ResultItem
	statistics []TaskData `json:"statistics"`
}

type TaskData struct {
	TaskId  string `json:"taskId"`
	Send    int    `json:"send"`
	Receive int    `json:"receive"`
	Display int    `json:"display"`
	Click   int    `json:"click"`
}
