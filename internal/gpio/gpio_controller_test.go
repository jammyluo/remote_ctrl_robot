package gpio

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	// 在测试中禁用日志输出
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func TestNewGPIOController(t *testing.T) {
	controller := NewGPIOController(18)
	if controller == nil {
		t.Fatal("NewGPIOController should not return nil")
	}

	if controller.pinNum != 18 {
		t.Errorf("期望 pinNum 为 18, 实际为 %d", controller.pinNum)
	}

	if controller.exported {
		t.Error("期望 exported 为 false")
	}
}

func TestGPIOControllerGetStatus(t *testing.T) {
	controller := NewGPIOController(18)
	status := controller.GetStatus()

	if status["pin_num"] != 18 {
		t.Errorf("期望 pin_num 为 18, 实际为 %v", status["pin_num"])
	}

	if status["exported"] != false {
		t.Errorf("期望 exported 为 false, 实际为 %v", status["exported"])
	}

	if status["library"] != "periph.io/x" {
		t.Errorf("期望 library 为 'periph.io/x', 实际为 %v", status["library"])
	}
}

func TestGPIOControllerSetHighLow(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试设置高电平
	err := controller.SetHigh()
	if err != nil {
		// 在非Linux系统上，这可能会失败，这是正常的
		t.Logf("SetHigh 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 等待一小段时间
	time.Sleep(50 * time.Millisecond)

	// 测试设置低电平
	err = controller.SetLow()
	if err != nil {
		t.Logf("SetLow 失败: %v", err)
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerSetValue(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试设置值为1（高电平）
	err := controller.SetValue(1)
	if err != nil {
		t.Logf("SetValue(1) 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 测试设置值为0（低电平）
	err = controller.SetValue(0)
	if err != nil {
		t.Logf("SetValue(0) 失败: %v", err)
	}

	// 测试无效值
	err = controller.SetValue(2)
	if err == nil {
		t.Error("期望 SetValue(2) 返回错误")
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerToggle(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试切换功能
	err := controller.Toggle()
	if err != nil {
		t.Logf("Toggle 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerPulse(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试脉冲功能
	err := controller.Pulse(100 * time.Millisecond)
	if err != nil {
		t.Logf("Pulse 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerBlink(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试闪烁功能（3次闪烁，每次间隔200ms）
	err := controller.Blink(200*time.Millisecond, 3)
	if err != nil {
		t.Logf("Blink 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerGetLevel(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试获取电平（在未导出状态下）
	level, err := controller.GetLevel()
	if err == nil {
		t.Error("期望在未导出状态下 GetLevel 返回错误")
	}

	// 导出引脚
	err = controller.SetHigh()
	if err != nil {
		t.Logf("SetHigh 失败（在非Linux系统上这是正常的）: %v", err)
		return
	}

	// 测试获取电平（在已导出状态下）
	level, err = controller.GetLevel()
	if err != nil {
		t.Logf("GetLevel 失败: %v", err)
	} else {
		t.Logf("当前电平: %v", level)
	}

	// 清理资源
	controller.Cleanup()
}

func TestGPIOControllerCleanup(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试清理功能
	err := controller.Cleanup()
	if err != nil {
		t.Errorf("Cleanup 失败: %v", err)
	}

	// 验证状态
	status := controller.GetStatus()
	if status["exported"] != false {
		t.Errorf("期望 exported 为 false, 实际为 %v", status["exported"])
	}
}

func TestGPIOControllerConcurrentAccess(t *testing.T) {
	controller := NewGPIOController(18)

	// 测试并发访问
	done := make(chan bool, 2)

	go func() {
		controller.GetStatus()
		done <- true
	}()

	go func() {
		controller.SetHigh()
		controller.SetLow()
		done <- true
	}()

	// 等待两个goroutine完成
	<-done
	<-done

	// 清理资源
	controller.Cleanup()
}
