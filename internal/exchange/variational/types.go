package variational

import "fmt"

// ErrorRes 错误响应
type ErrorRes struct {
	Message string `json:"error_message"` // 错误消息
}

// Error 实现error接口
func (err *ErrorRes) Error() string {
	return fmt.Sprintf("message: %s", err.Message)
}

// LoginRes 登录响应
type LoginRes struct {
	Token string `json:"token"`
}

// 生成签名数据响应
type GenerateSigningDataRes struct {
	Message string `json:"message"`
}
