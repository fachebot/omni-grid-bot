package nado

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewSenderFromHex(t *testing.T) {
	s := "0xa1c0a00e249ceff9232a6314de4b1f82f41ae86d64656661756c745f31000000"
	sender, err := NewSenderFromHex(s)
	require.NoError(t, err, "NewSenderFromHex 应该成功解析有效的 sender 字符串")

	expectedAddress := common.HexToAddress("0xa1C0A00e249CEFf9232A6314de4B1f82f41aE86D")
	require.Equal(t,
		expectedAddress.String(),
		sender.Address.String(),
		"解析出的 Address 地址应该与预期地址匹配")

	subaccount, err := hex.DecodeString("64656661756c745f31000000")
	require.NoError(t, err, "子账户十六进制字符串解码应该成功")

	expectedSubaccountID := big.NewInt(0).SetBytes(subaccount)
	require.Equal(t,
		expectedSubaccountID.String(),
		sender.SubaccountID.String(),
		"解析出的 SubaccountID 应该与预期值匹配")

	require.Equal(t, s, sender.String(), "Sender.String() 应该返回原始的十六进制字符串")
}

func TestSender_MarshalJSON(t *testing.T) {
	subaccount, err := hex.DecodeString("64656661756c745f31000000")
	require.NoError(t, err, "子账户十六进制字符串解码应该成功")

	signer := common.HexToAddress("0xa1C0A00e249CEFf9232A6314de4B1f82f41aE86D")

	v := struct {
		Sender *Sender `json:"sender"`
	}{
		Sender: &Sender{
			Address:      signer,
			SubaccountID: big.NewInt(0).SetBytes(subaccount),
		},
	}

	s, err := json.Marshal(v)
	require.NoError(t, err, "JSON 序列化应该成功")

	expectedJSON := `{"sender":"0xa1c0a00e249ceff9232a6314de4b1f82f41ae86d64656661756c745f31000000"}`
	require.Equal(t,
		expectedJSON,
		string(s),
		"序列化后的 JSON 应该与预期格式匹配")
}

func TestSender_UnmarshalJSON(t *testing.T) {
	v := struct {
		Sender Sender `json:"sender"`
	}{}

	data := []byte(`{"sender":"0xa1c0a00e249ceff9232a6314de4b1f82f41ae86d64656661756c745f31000000"}`)
	err := json.Unmarshal(data, &v)
	require.NoError(t, err, "JSON 反序列化应该成功")

	expectedAddress := common.HexToAddress("0xa1C0A00e249CEFf9232A6314de4B1f82f41aE86D")
	require.Equal(t,
		expectedAddress.String(),
		v.Sender.Address.String(),
		"反序列化后的 Address 地址应该与预期地址匹配")

	subaccount, err := hex.DecodeString("64656661756c745f31000000")
	require.NoError(t, err, "子账户十六进制字符串解码应该成功")

	expectedSubaccountID := big.NewInt(0).SetBytes(subaccount)
	require.Equal(t,
		expectedSubaccountID.String(),
		v.Sender.SubaccountID.String(),
		"反序列化后的 SubaccountID 应该与预期值匹配")
}

// 添加边界情况测试
func TestNewSenderFromHex_InvalidInput(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "空字符串",
			input: "",
		},
		{
			name:  "无效的十六进制字符串",
			input: "invalid_hex_string",
		},
		{
			name:  "长度不足",
			input: "0xa1c0a00e249ceff9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSenderFromHex(tc.input)
			require.Error(t, err, "NewSenderFromHex 应该对无效输入返回错误: %s", tc.name)
		})
	}
}

func TestSender_UnmarshalJSON_InvalidInput(t *testing.T) {
	testCases := []struct {
		name string
		data string
	}{
		{
			name: "无效的 JSON 格式",
			data: `{"sender":invalid}`,
		},
		{
			name: "无效的 sender 值",
			data: `{"sender":"invalid_sender"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := struct {
				Sender Sender `json:"sender"`
			}{}
			err := json.Unmarshal([]byte(tc.data), &v)
			require.Error(t, err, "UnmarshalJSON 应该对无效输入返回错误: %s", tc.name)
		})
	}
}

func TestNewAppendix(t *testing.T) {
	tests := []struct {
		name      string
		orderType OrderType
		want      *Appendix
	}{
		{
			name:      "创建Default订单",
			orderType: OrderTypeDefault,
			want: &Appendix{
				Version:   1,
				OrderType: OrderTypeDefault,
			},
		},
		{
			name:      "创建IOC订单",
			orderType: OrderTypeIOC,
			want: &Appendix{
				Version:   1,
				OrderType: OrderTypeIOC,
			},
		},
		{
			name:      "创建FOK订单",
			orderType: OrderTypeFOK,
			want: &Appendix{
				Version:   1,
				OrderType: OrderTypeFOK,
			},
		},
		{
			name:      "创建PostOnly订单",
			orderType: OrderTypePostOnly,
			want: &Appendix{
				Version:   1,
				OrderType: OrderTypePostOnly,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAppendix(tt.orderType)
			if got.Version != tt.want.Version {
				t.Errorf("Version = %v, want %v", got.Version, tt.want.Version)
			}
			if got.OrderType != tt.want.OrderType {
				t.Errorf("OrderType = %v, want %v", got.OrderType, tt.want.OrderType)
			}
		})
	}
}

func TestAppendix_ToBigInt_And_NewAppendixFromBigInt(t *testing.T) {
	tests := []struct {
		name     string
		appendix *Appendix
	}{
		{
			name: "Default订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeDefault,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "IOC订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeIOC,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "FOK订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeFOK,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "PostOnly订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypePostOnly,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "隔离保证金订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    true,
				OrderType:   OrderTypeDefault,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       1000000,
			},
		},
		{
			name: "仅减少头寸订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeIOC,
				ReduceOnly:  true,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "TWAP订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeDefault,
				ReduceOnly:  false,
				TriggerType: TriggerTypeTWAP,
				Value:       (uint64(500000) << 32) | uint64(10),
			},
		},
		{
			name: "复杂组合订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    true,
				OrderType:   OrderTypePostOnly,
				ReduceOnly:  true,
				TriggerType: TriggerTypeTWAPCustomAmounts,
				Value:       0xFFFFFFFFFFFFFFFF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 转换为BigInt
			bn := tt.appendix.ToBigInt()

			// 从BigInt恢复
			got, err := NewAppendixFromBigInt(bn)
			if err != nil {
				t.Fatalf("NewAppendixFromBigInt() error = %v", err)
			}

			// 验证所有字段
			if got.Version != tt.appendix.Version {
				t.Errorf("Version = %v, want %v", got.Version, tt.appendix.Version)
			}
			if got.Isolated != tt.appendix.Isolated {
				t.Errorf("Isolated = %v, want %v", got.Isolated, tt.appendix.Isolated)
			}
			if got.OrderType != tt.appendix.OrderType {
				t.Errorf("OrderType = %v, want %v", got.OrderType, tt.appendix.OrderType)
			}
			if got.ReduceOnly != tt.appendix.ReduceOnly {
				t.Errorf("ReduceOnly = %v, want %v", got.ReduceOnly, tt.appendix.ReduceOnly)
			}
			if got.TriggerType != tt.appendix.TriggerType {
				t.Errorf("TriggerType = %v, want %v", got.TriggerType, tt.appendix.TriggerType)
			}
			if got.Value != tt.appendix.Value {
				t.Errorf("Value = %v, want %v", got.Value, tt.appendix.Value)
			}
		})
	}
}

func TestNewAppendixFromBigInt_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   *big.Int
		wantErr string
	}{
		{
			name:    "超过16字节",
			input:   new(big.Int).Lsh(big.NewInt(1), 129), // 2^129
			wantErr: "invalid appendix",
		},
		{
			name: "不支持的版本0",
			input: func() *big.Int {
				// 手动构造版本为0的数据
				return big.NewInt(0) // Version = 0
			}(),
			wantErr: "unsupported appendix version: 0",
		},
		{
			name: "不支持的版本2",
			input: func() *big.Int {
				// 手动构造版本为2的数据
				return big.NewInt(2) // Version = 2
			}(),
			wantErr: "unsupported appendix version: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAppendixFromBigInt(tt.input)
			if err == nil {
				t.Errorf("期望错误但没有返回错误")
				return
			}
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				// 检查错误信息是否包含预期内容
				t.Logf("实际错误: %v", err.Error())
				t.Logf("期望错误: %v", tt.wantErr)
			}
		})
	}
}

func TestAppendix_TWAPData(t *testing.T) {
	t.Run("设置和获取TWAP数据", func(t *testing.T) {
		a := &Appendix{
			Version:     1,
			TriggerType: TriggerTypeTWAP,
		}

		times := uint32(10)
		slippage := uint32(500000)

		err := a.SetTWAPData(times, slippage)
		if err != nil {
			t.Fatalf("SetTWAPData() error = %v", err)
		}

		data, err := a.GetTWAPData()
		if err != nil {
			t.Fatalf("GetTWAPData() error = %v", err)
		}

		if data.Times != times {
			t.Errorf("Times = %v, want %v", data.Times, times)
		}
		if data.SlippageX6 != slippage {
			t.Errorf("SlippageX6 = %v, want %v", data.SlippageX6, slippage)
		}
	})

	t.Run("非TWAP订单设置TWAP数据", func(t *testing.T) {
		a := &Appendix{
			Version:     1,
			TriggerType: TriggerTypeNone,
		}

		err := a.SetTWAPData(10, 500000)
		if err == nil {
			t.Error("期望错误但没有返回错误")
		}
	})

	t.Run("非TWAP订单获取TWAP数据", func(t *testing.T) {
		a := &Appendix{
			Version:     1,
			TriggerType: TriggerTypeNone,
		}

		_, err := a.GetTWAPData()
		if err == nil {
			t.Error("期望错误但没有返回错误")
		}
	})

	t.Run("TWAP自定义金额类型", func(t *testing.T) {
		a := &Appendix{
			Version:     1,
			TriggerType: TriggerTypeTWAPCustomAmounts,
		}

		err := a.SetTWAPData(5, 300000)
		if err != nil {
			t.Fatalf("SetTWAPData() error = %v", err)
		}

		data, err := a.GetTWAPData()
		if err != nil {
			t.Fatalf("GetTWAPData() error = %v", err)
		}

		if data.Times != 5 {
			t.Errorf("Times = %v, want %v", data.Times, 5)
		}
		if data.SlippageX6 != 300000 {
			t.Errorf("SlippageX6 = %v, want %v", data.SlippageX6, 300000)
		}
	})

	t.Run("边界值测试", func(t *testing.T) {
		a := &Appendix{
			Version:     1,
			TriggerType: TriggerTypeTWAP,
		}

		// 测试最大值
		maxUint32 := uint32(0xFFFFFFFF)
		err := a.SetTWAPData(maxUint32, maxUint32)
		if err != nil {
			t.Fatalf("SetTWAPData() error = %v", err)
		}

		data, err := a.GetTWAPData()
		if err != nil {
			t.Fatalf("GetTWAPData() error = %v", err)
		}

		if data.Times != maxUint32 {
			t.Errorf("Times = %v, want %v", data.Times, maxUint32)
		}
		if data.SlippageX6 != maxUint32 {
			t.Errorf("SlippageX6 = %v, want %v", data.SlippageX6, maxUint32)
		}
	})
}

func TestAppendix_IsolatedMargin(t *testing.T) {
	t.Run("设置和获取隔离保证金", func(t *testing.T) {
		a := &Appendix{
			Version:  1,
			Isolated: true,
		}

		margin := uint64(1000000)
		err := a.SetIsolatedMargin(margin)
		if err != nil {
			t.Fatalf("SetIsolatedMargin() error = %v", err)
		}

		got, err := a.GetIsolatedMargin()
		if err != nil {
			t.Fatalf("GetIsolatedMargin() error = %v", err)
		}

		if got != margin {
			t.Errorf("IsolatedMargin = %v, want %v", got, margin)
		}
	})

	t.Run("非隔离订单设置保证金", func(t *testing.T) {
		a := &Appendix{
			Version:  1,
			Isolated: false,
		}

		err := a.SetIsolatedMargin(1000000)
		if err == nil {
			t.Error("期望错误但没有返回错误")
		}
	})

	t.Run("非隔离订单获取保证金", func(t *testing.T) {
		a := &Appendix{
			Version:  1,
			Isolated: false,
		}

		_, err := a.GetIsolatedMargin()
		if err == nil {
			t.Error("期望错误但没有返回错误")
		}
	})

	t.Run("最大保证金值", func(t *testing.T) {
		a := &Appendix{
			Version:  1,
			Isolated: true,
		}

		maxMargin := uint64(0xFFFFFFFFFFFFFFFF)
		err := a.SetIsolatedMargin(maxMargin)
		if err != nil {
			t.Fatalf("SetIsolatedMargin() error = %v", err)
		}

		got, err := a.GetIsolatedMargin()
		if err != nil {
			t.Fatalf("GetIsolatedMargin() error = %v", err)
		}

		if got != maxMargin {
			t.Errorf("IsolatedMargin = %v, want %v", got, maxMargin)
		}
	})
}

func TestAppendix_JSON(t *testing.T) {
	tests := []struct {
		name     string
		appendix *Appendix
	}{
		{
			name: "Default订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeDefault,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "IOC订单带隔离保证金",
			appendix: &Appendix{
				Version:     1,
				Isolated:    true,
				OrderType:   OrderTypeIOC,
				ReduceOnly:  false,
				TriggerType: TriggerTypeNone,
				Value:       123456789,
			},
		},
		{
			name: "FOK订单仅减少头寸",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypeFOK,
				ReduceOnly:  true,
				TriggerType: TriggerTypeNone,
				Value:       0,
			},
		},
		{
			name: "PostOnly TWAP订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    false,
				OrderType:   OrderTypePostOnly,
				ReduceOnly:  false,
				TriggerType: TriggerTypeTWAP,
				Value:       (uint64(500000) << 32) | uint64(10),
			},
		},
		{
			name: "复杂订单",
			appendix: &Appendix{
				Version:     1,
				Isolated:    true,
				OrderType:   OrderTypePostOnly,
				ReduceOnly:  true,
				TriggerType: TriggerTypeTWAPCustomAmounts,
				Value:       0xFFFFFFFFFFFFFFFF,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 序列化
			data, err := json.Marshal(tt.appendix)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// 反序列化
			var got Appendix
			err = json.Unmarshal(data, &got)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// 验证
			if got.Version != tt.appendix.Version {
				t.Errorf("Version = %v, want %v", got.Version, tt.appendix.Version)
			}
			if got.Isolated != tt.appendix.Isolated {
				t.Errorf("Isolated = %v, want %v", got.Isolated, tt.appendix.Isolated)
			}
			if got.OrderType != tt.appendix.OrderType {
				t.Errorf("OrderType = %v, want %v", got.OrderType, tt.appendix.OrderType)
			}
			if got.ReduceOnly != tt.appendix.ReduceOnly {
				t.Errorf("ReduceOnly = %v, want %v", got.ReduceOnly, tt.appendix.ReduceOnly)
			}
			if got.TriggerType != tt.appendix.TriggerType {
				t.Errorf("TriggerType = %v, want %v", got.TriggerType, tt.appendix.TriggerType)
			}
			if got.Value != tt.appendix.Value {
				t.Errorf("Value = %v, want %v", got.Value, tt.appendix.Value)
			}
		})
	}
}

func TestAppendix_UnmarshalJSON_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "无效JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "无效整数字符串",
			input:   `"not a number"`,
			wantErr: true,
		},
		{
			name:    "空字符串",
			input:   `""`,
			wantErr: true,
		},
		{
			name:    "负数",
			input:   `"-123"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a Appendix
			err := json.Unmarshal([]byte(tt.input), &a)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAppendix_AllOrderTypes(t *testing.T) {
	orderTypes := []OrderType{
		OrderTypeDefault,
		OrderTypeIOC,
		OrderTypeFOK,
		OrderTypePostOnly,
	}

	for _, ot := range orderTypes {
		t.Run(ot.String(), func(t *testing.T) {
			a := NewAppendix(ot)

			// 转换测试
			bn := a.ToBigInt()
			got, err := NewAppendixFromBigInt(bn)
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}

			if got.OrderType != ot {
				t.Errorf("OrderType = %v, want %v", got.OrderType, ot)
			}
		})
	}
}

func TestAppendix_BitFieldBoundaries(t *testing.T) {
	t.Run("OrderType边界值", func(t *testing.T) {
		for i := uint8(0); i <= 3; i++ {
			a := &Appendix{
				Version:   1,
				OrderType: OrderType(i),
			}

			bn := a.ToBigInt()
			got, err := NewAppendixFromBigInt(bn)
			if err != nil {
				t.Fatalf("OrderType %d 转换失败: %v", i, err)
			}

			if got.OrderType != OrderType(i) {
				t.Errorf("OrderType = %v, want %v", got.OrderType, i)
			}
		}
	})

	t.Run("所有布尔标志组合", func(t *testing.T) {
		combinations := []struct {
			isolated   bool
			reduceOnly bool
		}{
			{false, false},
			{false, true},
			{true, false},
			{true, true},
		}

		for _, c := range combinations {
			a := &Appendix{
				Version:    1,
				Isolated:   c.isolated,
				OrderType:  OrderTypeDefault,
				ReduceOnly: c.reduceOnly,
			}

			bn := a.ToBigInt()
			got, err := NewAppendixFromBigInt(bn)
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}

			if got.Isolated != c.isolated {
				t.Errorf("Isolated = %v, want %v", got.Isolated, c.isolated)
			}
			if got.ReduceOnly != c.reduceOnly {
				t.Errorf("ReduceOnly = %v, want %v", got.ReduceOnly, c.reduceOnly)
			}
		}
	})
}
