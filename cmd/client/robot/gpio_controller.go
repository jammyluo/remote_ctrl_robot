package robot

import (
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/warthog618/go-gpiocdev"
)

// GPIOController GPIO控制器
type GPIOController struct {
	chip    *gpiocdev.Chip
	line    *gpiocdev.Line
	pinNum  int
	exported bool
}

// NewGPIOController 创建新的GPIO控制器
func NewGPIOController(pinNum int) *GPIOController {
	return &GPIOController{
		pinNum:   pinNum,
		exported: false,
	}
}

// exportPin 导出GPIO引脚
func (g *GPIOController) exportPin() error {
	if g.exported {
		return nil
	}

	// 打开GPIO芯片 (通常是gpiochip0)
	chip, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
		return fmt.Errorf("failed to open GPIO chip: %v", err)
	}
	g.chip = chip

	// 获取GPIO线
	line, err := chip.RequestLine(g.pinNum, gpiocdev.AsOutput(0))
	if err != nil {
		chip.Close()
		return fmt.Errorf("failed to get GPIO line %d: %v", g.pinNum, err)
	}
	g.line = line

	
	// 初始状态设为低电平
	if err := line.SetValue(0); err != nil {
		line.Close()
		chip.Close()
		return fmt.Errorf("failed to set initial value for GPIO line %d: %v", g.pinNum, err)
	}

	g.exported = true
	log.Info().Int("pin", g.pinNum).Msg("GPIO pin exported and set to output (gpiocdev)")
	return nil
}

// setPinValue 设置引脚值
func (g *GPIOController) setPinValue(value int) error {
	if !g.exported {
		if err := g.exportPin(); err != nil {
			return err
		}
	}

	if err := g.line.SetValue(value); err != nil {
		return fmt.Errorf("failed to set GPIO line %d value to %d: %v", g.pinNum, value, err)
	}

	return nil
}

// BuzzerOn 打开蜂鸣器
func (g *GPIOController) BuzzerOn() error {
	return g.setPinValue(1)
}

// BuzzerOff 关闭蜂鸣器
func (g *GPIOController) BuzzerOff() error {
	return g.setPinValue(0)
}

// StartBuzzerPattern 开始蜂鸣器模式
func (g *GPIOController) StartBuzzerPattern() {
	log.Info().Int("pin", g.pinNum).Msg("Starting buzzer pattern (gpiocdev)")

	go func() {
		for {
			// 关闭蜂鸣器
			if err := g.BuzzerOff(); err != nil {
				log.Error().Err(err).Msg("Failed to turn off buzzer")
				continue
			}
			// 等待0.06秒
			time.Sleep(60 * time.Millisecond)
			
			// 打开蜂鸣器
			if err := g.BuzzerOn(); err != nil {
				log.Error().Err(err).Msg("Failed to turn on buzzer")
				continue
			}
			log.Debug().Msg("Buzzer on")
		}
	}()
}

// StopBuzzerPattern 停止蜂鸣器模式
func (g *GPIOController) StopBuzzerPattern() {
	log.Info().Int("pin", g.pinNum).Msg("Stopping buzzer pattern (gpiocdev)")
	// 关闭蜂鸣器
	g.BuzzerOff()
}

// Cleanup 清理GPIO资源
func (g *GPIOController) Cleanup() error {
	if !g.exported {
		return nil
	}

	// 先关闭蜂鸣器
	g.BuzzerOff()

	// 关闭GPIO线和芯片
	if g.line != nil {
		g.line.Close()
	}
	if g.chip != nil {
		g.chip.Close()
	}

	g.exported = false
	log.Info().Int("pin", g.pinNum).Msg("GPIO pin cleaned up (gpiocdev)")
	return nil
} 