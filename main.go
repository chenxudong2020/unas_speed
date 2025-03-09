package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/d2r2/go-i2c"
	"github.com/shirou/gopsutil/v3/host"
)

const (
	FAN_SPEED_70_UP   = 255
	FAN_SPEED_65_75   = 200
	FAN_SPEED_55_65   = 150
	FAN_SPEED_55_DOWN = 100
	I2C_ADDR          = 0x54
	I2C_BUS           = 0
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
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return 0, err
	}

	for _, sensor := range sensors {
		if sensor.SensorKey == "coretemp_packageid0" {
			return sensor.Temperature, nil
		}
	}
	return 0, fmt.Errorf("CPU temperature not found")
}

func setFanSpeed(speed int) {
	i2c, err := i2c.NewI2C(I2C_ADDR, I2C_BUS)
	if err != nil {
		eslog("Error initializing I2C:", err)
		return
	}
	defer i2c.Close()

	_, err = i2c.WriteBytes([]byte{0xf0, byte(speed)})
	if err != nil {
		eslog("Error setting fan speed:", err)
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

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

	<-sigs
	tempatureOff()
}
