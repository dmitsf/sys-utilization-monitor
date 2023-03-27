package main

import (
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/shirou/gopsutil/cpu"
	"log"
	"time"
)

func main() {
	for {
		//Percent calculates the percentage of cpu used either per CPU or combined.
		percent, _ := cpu.Percent(time.Second, true)
		fmt.Printf("--> %f\n", percent)
		/*fmt.Printf("  User: %.2f\n", percent[cpu.CPUser])
		fmt.Printf("  Nice: %.2f\n", percent[cpu.CPNice])
		fmt.Printf("   Sys: %.2f\n", percent[cpu.CPSys])
		fmt.Printf("  Intr: %.2f\n", percent[cpu.CPIntr])
		fmt.Printf("  Idle: %.2f\n", percent[cpu.CPIdle])
		fmt.Printf("States: %.2f\n", percent[cpu.CPUStates])*/
		time.Sleep(time.Second)
		//}

		ret := nvml.Init()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
		}
		defer func() {
			ret := nvml.Shutdown()
			if ret != nvml.SUCCESS {
				log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
			}
		}()

		count, ret := nvml.DeviceGetCount()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
		}

		for i := 0; i < count; i++ {
			device, ret := nvml.DeviceGetHandleByIndex(i)
			if ret != nvml.SUCCESS {
				log.Fatalf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
			}

			uuid, ret := device.GetUUID()
			if ret != nvml.SUCCESS {
				log.Fatalf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
			}

			fmt.Printf("%v\n", uuid)
		}
	}
}
