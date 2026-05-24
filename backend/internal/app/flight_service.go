package app

import (
	"context"
	"runtime"
	"syscall"
	"time"

	"connectrpc.com/connect"
	diagnosticsv1 "github.com/index/stint/backend/gen/api/diagnostics/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DiagnosticsService struct {
	startedAt time.Time
}

func NewDiagnosticsService() *DiagnosticsService {
	return &DiagnosticsService{
		startedAt: time.Now(),
	}
}

func (s *DiagnosticsService) Probe(
	_ context.Context,
	req *connect.Request[diagnosticsv1.ProbeRequest],
) (*connect.Response[diagnosticsv1.ProbeResponse], error) {
	response := &diagnosticsv1.ProbeResponse{
		ClientSentAt:       req.Msg.GetClientSentAt(),
		ServerProcessedAt:  timestamppb.Now(),
		CpuPercent:         s.cpuPercent(),
		BackendMemoryBytes: memoryBytes(),
		UptimeSeconds:      uint64(time.Since(s.startedAt).Seconds()),
	}

	return connect.NewResponse(response), nil
}

func (s *DiagnosticsService) cpuPercent() float32 {
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
