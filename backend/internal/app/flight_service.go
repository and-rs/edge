package app

import (
	"context"
	"runtime"
	"syscall"
	"time"

	"connectrpc.com/connect"
	flightv1 "github.com/index/stint/backend/gen/api/flight/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FlightService struct {
	startedAt time.Time
}

func NewFlightService() *FlightService {
	return &FlightService{
		startedAt: time.Now(),
	}
}

func (s *FlightService) Probe(
	_ context.Context,
	req *connect.Request[flightv1.ProbeRequest],
) (*connect.Response[flightv1.ProbeResponse], error) {
	response := &flightv1.ProbeResponse{
		ClientSentAt:       req.Msg.GetClientSentAt(),
		ServerProcessedAt:  timestamppb.Now(),
		CpuPercent:         s.cpuPercent(),
		BackendMemoryBytes: memoryBytes(),
		UptimeSeconds:      uint64(time.Since(s.startedAt).Seconds()),
	}

	return connect.NewResponse(response), nil
}

func (s *FlightService) cpuPercent() float32 {
	var usage syscall.Rusage
	err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	if err != nil {
		return 0
	}

	cpuTime :=
		time.Duration(usage.Utime.Sec)*time.Second +
			time.Duration(usage.Utime.Usec)*time.Microsecond +
			time.Duration(usage.Stime.Sec)*time.Second +
			time.Duration(usage.Stime.Usec)*time.Microsecond

	elapsedSeconds := time.Since(s.startedAt).Seconds()
	if elapsedSeconds <= 0 {
		return 0
	}

	cpuPercent := float32(cpuTime.Seconds()/elapsedSeconds) * 100
	maxPercent := float32(runtime.NumCPU() * 100)
	if cpuPercent > maxPercent {
		return maxPercent
	}
	return cpuPercent
}

func memoryBytes() uint64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats.Alloc
}
