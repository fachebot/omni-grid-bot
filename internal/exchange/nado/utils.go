package nado

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Sender struct {
	Address      common.Address
	SubaccountID *big.Int
}

// String 方法用于将 Sender 转换为字符串
func (s *Sender) String() string {
	var buf [32]byte

	var bytes12 [12]byte
	copy(bytes12[:], s.SubaccountID.Bytes())

	copy(buf[:], s.Address[:])
	copy(buf[len(s.Address):], bytes12[:])
	return "0x" + hex.EncodeToString(buf[:])
}

// NewSenderFromHex 从hex创建Sender
func NewSenderFromHex(s string) (Sender, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}

	data, err := hex.DecodeString(s)
	if err != nil {
		return Sender{}, err
	}

	if len(data) != 32 {
		return Sender{}, ErrInvalidSender
	}

	sender := Sender{
		Address:      common.BytesToAddress(data[:common.AddressLength]),
		SubaccountID: big.NewInt(0).SetBytes(data[common.AddressLength:]),
	}
	return sender, nil
}

// MarshalJSON 实现 JSON 序列化接口
func (s Sender) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON 实现 JSON 反序列化接口
func (s *Sender) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	sender, err := NewSenderFromHex(str)
	if err != nil {
		return err
	}
	*s = sender
	return nil
}

// OrderType 订单执行类型
type OrderType uint8

const (
	OrderTypeDefault  OrderType = 0 // 标准限价单
	OrderTypeIOC      OrderType = 1 // 立即成交或取消
	OrderTypeFOK      OrderType = 2 // 全部成交或取消
	OrderTypePostOnly OrderType = 3 // 仅提供流动性
)

func (ot OrderType) String() string {
	switch ot {
	case OrderTypeDefault:
		return "Default"
	case OrderTypeIOC:
		return "IOC"
	case OrderTypeFOK:
		return "FOK"
	case OrderTypePostOnly:
		return "PostOnly"
	default:
		return ""
	}
}

// TriggerType 触发类型
type TriggerType uint8

const (
	TriggerTypeNone              TriggerType = 0 // 立即执行
	TriggerTypePrice             TriggerType = 1 // 基于价格的条件单
	TriggerTypeTWAP              TriggerType = 2 // 时间加权平均价格
	TriggerTypeTWAPCustomAmounts TriggerType = 3 // TWAP自定义数量
)

// TWAPData TWAP配置数据
type TWAPData struct {
	Times      uint32 // 32-bits: TWAP执行次数
	SlippageX6 uint32 // 32-bits: 最大滑点 × 1,000,000
}

// Appendix 订单附录结构体 (128-bit)
// 文档：https://docs.nado.xyz/developer-resources/api/order-appendix
type Appendix struct {
	Version     uint8       // 8-bits (0-7): 协议版本
	Isolated    bool        // 1-bit (8): 是否使用隔离保证金
	OrderType   OrderType   // 2-bits (9-10): 订单执行类型
	ReduceOnly  bool        // 1-bit (11): 仅减少头寸
	TriggerType TriggerType // 2-bits (12-13): 触发类型
	Reserved    struct{}    // 50-bits (14-63): 保留位，必须为0
	Value       uint64      // 64-bits (64-127): 上下文数据
}

// NewAppendix 创建新的Appendix
func NewAppendix(orderType OrderType) *Appendix {
	return &Appendix{
		Version:   1,
		OrderType: orderType,
	}
}

// NewAppendixFromBigInt 从big.Int创建新的Appendix
func NewAppendixFromBigInt(bn *big.Int) (Appendix, error) {
	buf := bn.Bytes()
	if len(buf) > 16 {
		return Appendix{}, errors.New("invalid appendix")
	}

	data := make([]byte, 16)
	copy(data[len(data)-len(buf):], buf)

	// 转换字节序列
	slices.Reverse(data)

	// 将[16]byte转换为uint128 (小端序)
	// 低64位
	low := binary.LittleEndian.Uint64(data[0:8])
	// 高64位
	high := binary.LittleEndian.Uint64(data[8:16])

	appendix := Appendix{
		Version:     uint8(low & 0xFF),               // bits 0-7
		Isolated:    (low & (1 << 8)) != 0,           // bit 8
		OrderType:   OrderType((low >> 9) & 0b11),    // bits 9-10
		ReduceOnly:  (low & (1 << 11)) != 0,          // bit 11
		TriggerType: TriggerType((low >> 12) & 0b11), // bits 12-13
		Value:       high,                            // bits 64-127
	}

	// 验证版本
	if appendix.Version != 1 {
		return Appendix{}, fmt.Errorf("unsupported appendix version: %d", appendix.Version)
	}

	return appendix, nil
}

// ToBigInt 转换为big.Int类型
func (a *Appendix) ToBigInt() *big.Int {
	var result [16]byte

	// 构建低64位
	low := uint64(a.Version) // bits 0-7

	if a.Isolated {
		low |= 1 << 8 // bit 8
	}

	low |= (uint64(a.OrderType) & 0b11) << 9 // bits 9-10

	if a.ReduceOnly {
		low |= 1 << 11 // bit 11
	}

	low |= (uint64(a.TriggerType) & 0b11) << 12 // bits 12-13
	// bits 14-63 保留，设为0

	// 写入低64位
	binary.LittleEndian.PutUint64(result[0:8], low)

	// 写入高64位 (Value)
	binary.LittleEndian.PutUint64(result[8:16], a.Value)

	// 转换字节序列
	slices.Reverse(result[:])

	return big.NewInt(0).SetBytes(result[:])
}

// GetTWAPData 提取TWAP数据
func (a *Appendix) GetTWAPData() (*TWAPData, error) {
	if a.TriggerType != TriggerTypeTWAP &&
		a.TriggerType != TriggerTypeTWAPCustomAmounts {
		return nil, fmt.Errorf("not a TWAP order")
	}

	return &TWAPData{
		Times:      uint32(a.Value & 0xFFFFFFFF),         // 低32位
		SlippageX6: uint32((a.Value >> 32) & 0xFFFFFFFF), // 高32位
	}, nil
}

// SetTWAPData 设置TWAP数据
func (a *Appendix) SetTWAPData(times uint32, slippageX6 uint32) error {
	if a.TriggerType != TriggerTypeTWAP &&
		a.TriggerType != TriggerTypeTWAPCustomAmounts {
		return fmt.Errorf("not a TWAP order")
	}

	a.Value = (uint64(slippageX6) << 32) | uint64(times)
	return nil
}

// GetIsolatedMargin 提取隔离保证金
func (a *Appendix) GetIsolatedMargin() (uint64, error) {
	if !a.Isolated {
		return 0, fmt.Errorf("not an isolated order")
	}
	return a.Value, nil
}

// SetIsolatedMargin 设置隔离保证金
func (a *Appendix) SetIsolatedMargin(marginX6 uint64) error {
	if !a.Isolated {
		return fmt.Errorf("not an isolated order")
	}
	a.Value = marginX6
	return nil
}

// MarshalJSON 实现 JSON 序列化接口
func (a Appendix) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.ToBigInt().String())
}

// UnmarshalJSON 实现 JSON 反序列化接口
func (s *Appendix) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	bn, ok := big.NewInt(0).SetString(str, 10)
	if !ok {
		return errors.New("invalid integer")
	}

	appendix, err := NewAppendixFromBigInt(bn)
	if err != nil {
		return err
	}
	*s = appendix
	return nil
}

// GenerateNonceWithRandom 生成订单 nonce (指定随机数)
// discardAfterMs: 订单在多少毫秒后应该被忽略
// randomInt: 指定的随机整数 (0 到 1048575)
// 返回值: nonce 字符串
func GenerateNonceWithRandom(discardAfterMs int64, randomInt int64) uint64 {
	currentTimeMs := time.Now().UnixMilli()
	recvTime := currentTimeMs + discardAfterMs
	nonce := (recvTime << 20) + randomInt

	return uint64(nonce)
}
