package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/55gY/new-api-lite/common"
	"github.com/55gY/new-api-lite/constant"
	"github.com/55gY/new-api-lite/logger"
	relaycommon "github.com/55gY/new-api-lite/relay/common"
	"github.com/55gY/new-api-lite/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/gin-gonic/gin"
)

const (
	InitialScannerBufferSize    = 64 << 10 // 64KB (64*1024)
	DefaultMaxScannerBufferSize = 64 << 20 // 64MB (64*1024) default SSE buffer size
	DefaultPingInterval         = 10 * time.Second
)

func getScannerBufferSize() int {
	if constant.StreamScannerMaxBufferMB > 0 {
		return constant.StreamScannerMaxBufferMB << 20
	}
	return DefaultMaxScannerBufferSize
}

func StreamScannerHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo, dataHandler func(data string, sr *StreamResult)) {
	if resp == nil || resp.Body == nil || dataHandler == nil {
		return
	}

	if info.StreamStatus == nil {
		info.StreamStatus = relaycommon.NewStreamStatus()
	}

	defer resp.Body.Close()

	streamingTimeout := time.Duration(constant.StreamingTimeout) * time.Second
	if streamingTimeout <= 0 {
		streamingTimeout = DefaultPingInterval
	}

	requestCtx := context.Background()
	if c != nil && c.Request != nil {
		requestCtx = c.Request.Context()
	}
	ctx, cancel := context.WithCancel(requestCtx)
	defer cancel()

	var (
		scanner    = bufio.NewScanner(resp.Body)
		ticker     = time.NewTicker(streamingTimeout)
		pingTicker *time.Ticker
		writeMutex sync.Mutex
		wg         sync.WaitGroup
	)
	defer ticker.Stop()

	generalSettings := operation_setting.GetGeneralSetting()
	pingEnabled := generalSettings.PingIntervalEnabled && !info.DisablePing
	pingInterval := time.Duration(generalSettings.PingIntervalSeconds) * time.Second
	if pingInterval <= 0 {
		pingInterval = DefaultPingInterval
	}
	if pingEnabled {
		pingTicker = time.NewTicker(pingInterval)
		defer pingTicker.Stop()
	}

	logger.LogDebug(c, "relay timeout seconds: %d", common.RelayTimeout)
	logger.LogDebug(c, "relay max idle conns: %d", common.RelayMaxIdleConns)
	logger.LogDebug(c, "relay max idle conns per host: %d", common.RelayMaxIdleConnsPerHost)
	logger.LogDebug(c, "streaming timeout seconds: %d", int64(streamingTimeout.Seconds()))
	logger.LogDebug(c, "ping interval seconds: %d", int64(pingInterval.Seconds()))

	scanner.Buffer(make([]byte, InitialScannerBufferSize), getScannerBufferSize())
	scanner.Split(bufio.ScanLines)
	SetEventStreamHeaders(c)

	dataChan := make(chan string, 10)
	scannerDone := make(chan struct{})

	if pingEnabled && pingTicker != nil {
		wg.Add(1)
		gopool.Go(func() {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					err := fmt.Errorf("ping panic: %v", r)
					logger.LogError(c, err.Error())
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, err)
					cancel()
				}
				logger.LogDebug(c, "ping goroutine exited")
			}()

			pingTimeout := time.NewTimer(30 * time.Minute)
			defer pingTimeout.Stop()

			for {
				select {
				case <-pingTicker.C:
					writeMutex.Lock()
					err := PingData(c)
					writeMutex.Unlock()
					if err != nil {
						logger.LogError(c, "ping data error: "+err.Error())
						info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPingFail, err)
						cancel()
						return
					}
					logger.LogDebug(c, "ping data sent")
				case <-scannerDone:
					return
				case <-ctx.Done():
					return
				case <-pingTimeout.C:
					logger.LogError(c, "ping goroutine max duration reached")
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPingFail, fmt.Errorf("ping max duration reached"))
					cancel()
					return
				}
			}
		})
	}

	wg.Add(1)
	gopool.Go(func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				err := fmt.Errorf("data handler panic: %v", r)
				logger.LogError(c, err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, err)
				cancel()
			}
		}()

		sr := newStreamResult(info.StreamStatus)
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-dataChan:
				if !ok {
					return
				}
				sr.reset()
				writeMutex.Lock()
				dataHandler(data, sr)
				writeMutex.Unlock()
				if sr.IsStopped() {
					cancel()
					return
				}
			}
		}
	})

	wg.Add(1)
	gopool.Go(func() {
		defer func() {
			close(dataChan)
			close(scannerDone)
			wg.Done()
			if r := recover(); r != nil {
				err := fmt.Errorf("scanner panic: %v", r)
				logger.LogError(c, err.Error())
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonPanic, err)
				cancel()
			}
			logger.LogDebug(c, "scanner goroutine exited")
		}()

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			ticker.Reset(streamingTimeout)
			data := scanner.Text()
			logger.LogDebug(c, "stream scanner data: %s", data)

			if len(data) < 6 {
				continue
			}
			if data[:5] != "data:" && data[:6] != "[DONE]" {
				continue
			}
			data = strings.TrimSpace(data[5:])
			if data == "" {
				continue
			}
			if !strings.HasPrefix(data, "[DONE]") {
				info.SetFirstResponseTime()
				info.ReceivedResponseCount++

				select {
				case dataChan <- data:
				case <-ctx.Done():
					return
				}
				continue
			}

			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
			logger.LogDebug(c, "received [DONE], stopping scanner")
			return
		}

		if err := scanner.Err(); err != nil && err != io.EOF && ctx.Err() == nil {
			logger.LogError(c, "scanner error: "+err.Error())
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonScannerErr, err)
			return
		}
		if ctx.Err() == nil {
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonEOF, nil)
		}
	})

	stopRequested := false
	select {
	case <-ticker.C:
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonTimeout, nil)
		stopRequested = true
	case <-requestCtx.Done():
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, requestCtx.Err())
		stopRequested = true
	case <-ctx.Done():
		stopRequested = true
	case <-scannerDone:
		stopRequested = ctx.Err() != nil
	}

	if stopRequested {
		cancel()
		// Closing the upstream response body unblocks a Scanner waiting in Read.
		_ = resp.Body.Close()
	}
	wg.Wait()

	if info.StreamStatus.IsNormalEnd() && !info.StreamStatus.HasErrors() {
		logger.LogInfo(c, fmt.Sprintf("stream ended: %s", info.StreamStatus.Summary()))
	} else {
		logger.LogError(c, fmt.Sprintf("stream ended: %s, received=%d", info.StreamStatus.Summary(), info.ReceivedResponseCount))
	}
}
