package main

import (
	"context"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"math"
	"time"
)

const startingIntervalMilliseconds = 10000 * time.Millisecond
const measurementIntervalMilliseconds = 1000 * time.Millisecond

type systemUtilizationEvent struct {
	message  string
	increase float64
}

type getUtilizationFn func() (float64, error)

type systemUtilizationWatcher struct {
	startingIntervalMilliseconds    time.Duration
	measurementIntervalMilliseconds time.Duration
	stdIncreaseThreshold            float64
	strictThreshold                 float64
	getUtilizationFn                getUtilizationFn
}

/*
func (s *systemUtilizationWatcher) watch(ctx context.Context)  {
	//logger := logging.FromContext(ctx)
	//logger.Info("starting watch")

	outChan := make(chan systemUtilizationEvent)
	defer close(outChan)
	errChan := make(chan error)
	defer close(errChan)

	go s.watchLoop(outChan, errChan)

	return outChan, errChan
}*/

func main() {
	systemUtilizationMonitor := func(ctx context.Context) {
		outChan := make(chan systemUtilizationEvent)
		defer close(outChan)
		errChan := make(chan error)
		defer close(errChan)

		getUtilizationFnCPU := func() (float64, error) {
			cpuPercent, err := cpu.Percent(time.Second, false)
			if err != nil {
				return 0, fmt.Errorf("error getting CPU utilization %w", err)
			}
			return cpuPercent[0], nil
		}

		suw := systemUtilizationWatcher{
			startingIntervalMilliseconds:    startingIntervalMilliseconds,
			measurementIntervalMilliseconds: measurementIntervalMilliseconds,
			stdIncreaseThreshold:            1,
			getUtilizationFn:                getUtilizationFnCPU,
		}

		go suw.watchLoop(ctx, outChan, errChan)

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errChan:
				// do not break of panic on error.
				fmt.Printf("Error: %v", err)
			case event := <-outChan:
				fmt.Printf("%s: %f stdDev", event.message, event.increase)
			}
		}
	}

	ctx := context.Background()
	go systemUtilizationMonitor(ctx)

	time.Sleep(100 * time.Second)

	fmt.Println("ready")
	/*for {
		//Percent calculates the percentage of cpu used either per CPU or combined.
		percent, _ := cpu.Percent(time.Second, true)
		percent2, _ := cpu.Percent(time.Second, false)

		fmt.Printf("--> %f  %f\n", percent, percent2)

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

				//uuid, ret := device.GetUUID()
				//if ret != nvml.SUCCESS {
				//	log.Fatalf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
				//}
				t, s := device.GetUtilizationRates()
				fmt.Printf("%v %v\n", t, s)

			}
	}*/
}

func (s *systemUtilizationWatcher) watchLoop(ctx context.Context, outChan chan systemUtilizationEvent, errChan chan error) {
	measurementsSum := .0
	var measurementsSample []float64

	// Get measurements asynchronously

	ticker := time.NewTicker(s.measurementIntervalMilliseconds)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				utilization, err := s.getUtilizationFn()

				if err != nil {
					errChan <- fmt.Errorf("error while getting utilization %s", err)
					return
				}

				measurementsSample = append(measurementsSample, utilization)
				measurementsSum += utilization
			}
		}
	}()

	time.Sleep(s.startingIntervalMilliseconds)
	ticker.Stop()
	done <- true

	measurementsNumber := len(measurementsSample)

	// Get mean of the sample
	mean := measurementsSum / float64(measurementsNumber)

	devSum := .0
	for i := 0; i < len(measurementsSample); i++ {
		devSum += math.Pow(measurementsSample[i]-mean, 2)
	}
	// Get unbiased sample variance
	variance := (1.0 / float64(measurementsNumber-1)) * devSum
	stdDev := math.Sqrt(variance)

	fmt.Printf("-----> %v", measurementsSample)

	ticker = time.NewTicker(s.measurementIntervalMilliseconds)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			utilization, err := s.getUtilizationFn()

			if err != nil {
				errChan <- fmt.Errorf("error while getting utilization %s", err)
			}

			fmt.Printf("--> %f  \n", utilization)

			if utilization > mean+s.stdIncreaseThreshold*stdDev {
				outChan <- systemUtilizationEvent{
					message:  fmt.Sprintf("utilization spike detected, standard deviation threshold exceeded"),
					increase: (utilization - (mean + stdDev)) / stdDev,
				}
			} else if s.strictThreshold > 0 && utilization > s.strictThreshold {
				outChan <- systemUtilizationEvent{
					message:  fmt.Sprintf("utilization spike detected, strict threshold exceeded"),
					increase: utilization - s.strictThreshold,
				}
			}
		}
	}
}
