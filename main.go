package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/periph/host"
	"github.com/periph/host/i2c"
)

const (
	FAN_SPEED_70_UP   = 255
	FAN_SPEED_65_75   = 200
	FAN_SPEED_55_65   = 150
	FAN_SPEED_55_DOWN = 100
	I2C_ADDR          = 0x54
)

var (
	prevCPUTemp  float64
	prevFanSpeed int
	isDebug      = true
)

func getFanSpeedForTemp(cpuTemp float64) int {
	switch {
	case cpuTemp >= 70:
		return FAN_SPEED_70_UP
	case cpuTemp >= 65 && cpuTemp < 75:
		return FAN_SPEED_65_75
	case cpuTemp >= 55 && cpuTemp < 65:
		return FAN_SPEED_55_65
	default:
		return FAN_SPEED_55_DOWN
	}
}

func eslog(v ...interface{}) {
	if isDebug {
		fmt.Println(v...)
	}
}

func tempatureOff() {
	eslog("关闭tempature并设置风扇转速100")
	setFanSpeed(100)
	os.Exit(0)
}

func getCPUTemp() (float64, error) {
	eslog("Attempting to get CPU temperature...")

	sensors, err := host.SensorsTemperatures()
	if err != nil {
		eslog("Error retrieving sensor temperatures:", err)
		return 0, err
	}

	eslog(fmt.Sprintf("Number of sensors found: %d", len(sensors)))

	if len(sensors) == 0 {
		eslog("No sensors found")
		return 0, fmt.Errorf("no sensors found")
	}

	for _, sensor := range sensors {
		eslog(fmt.Sprintf("Sensor: %s, Temperature: %.2f", sensor.SensorKey, sensor.Temperature))
		if sensor.Temperature > 0 {
			return sensor.Temperature, nil
		}
	}
	return 0, fmt.Errorf("CPU temperature not found")
}

func setFanSpeed(speed int) {
	eslog(fmt.Sprintf("Setting fan speed to %d", speed))

	// 初始化 periph 库
	if _, err := host.Init(); err != nil {
		eslog("Error initializing periph:", err)
		return
	}

	// 打开 I2C 总线
	bus, err := i2c.New("/dev/i2c-0")
	if err != nil {
		eslog("Error opening I2C bus:", err)
		return
	}

	// 创建 I2C 设备
	dev := &i2c.Dev{Addr: I2C_ADDR, Bus: bus}

	// 设置风扇速度
	err = dev.Tx([]byte{0xf0, byte(speed)}, nil)
	if err != nil {
		eslog("Error setting fan speed:", err)
	} else {
		eslog("Fan speed set successfully")
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	eslog("Starting main function...")

	setFanSpeed(255)

	go func() {
		for {
			select {
			case <-ticker.C:
				cpuTemp, err := getCPUTemp()
				if err != nil {
					eslog("Error getting CPU temperature:", err)
					continue
				}

				eslog(fmt.Sprintf("当前CPU温度: %.2f", cpuTemp))

				currentFanSpeed := getFanSpeedForTemp(cpuTemp)

				if prevCPUTemp != 0 && prevFanSpeed != currentFanSpeed {
					eslog(fmt.Sprintf("CPU温度从 %.2f 变为 %.2f，挡位从 %d 变为 %d", prevCPUTemp, cpuTemp, prevFanSpeed, currentFanSpeed))
					setFanSpeed(currentFanSpeed)
				}
				prevCPUTemp = cpuTemp
				prevFanSpeed = currentFanSpeed
			}
		}
	}()

	eslog("Waiting for termination signal...")
	<-sigs
	tempatureOff()
}
