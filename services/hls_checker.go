package services

import (
	"context"
	"fmt"
	"io"
	"iptv-aggregator/utils"
	"net/http"
	"strings"
	"time"
)

// HLSChecker 处理 HLS 相关的检测逻辑
type HLSChecker struct {
	client        *http.Client
	deepCheck     bool
	loopChecks    int
	loopInterval  time.Duration
	segmentWindow int
	logger        *utils.Logger
}

// NewHLSChecker 创建新的 HLS 检查器
func NewHLSChecker(client *http.Client, deepCheck bool, loopChecks int, loopInterval time.Duration, segmentWindow int, logger *utils.Logger) *HLSChecker {
	return &HLSChecker{
		client:        client,
		deepCheck:     deepCheck,
		loopChecks:    loopChecks,
		loopInterval:  loopInterval,
		segmentWindow: segmentWindow,
		logger:        logger,
	}
}

// CheckM3U8 检查 M3U8 播放列表
func (hc *HLSChecker) CheckM3U8(ctx context.Context, rawURL string, userAgent string) (bool, int64, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := hc.client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, 0, fmt.Errorf("M3U8下载失败: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}

	m3u8Text := string(body)
	tsURLs := extractTSURLs(rawURL, m3u8Text)

	if len(tsURLs) == 0 {
		return false, 0, fmt.Errorf("未找到TS分片")
	}

	// 只要有一个分片可用即认为可用
	// 限制最多测试 3 个 TS 分片，每个最多 5 秒超时
	headOK := false
	maxTSRequests := 3
	testedCount := 0

	for _, tsURL := range tsURLs {
		if testedCount >= maxTSRequests {
			break
		}

		// 为每个 TS 请求创建独立的超时上下文（5 秒），但继承父 Context 的取消信号
		tsCtx, tsCancel := context.WithTimeout(ctx, 5*time.Second)
		tsReq, _ := http.NewRequestWithContext(tsCtx, "HEAD", tsURL, nil)
		tsReq.Header.Set("User-Agent", userAgent)

		tsResp, err := hc.client.Do(tsReq)
		if err == nil {
			tsResp.Body.Close()
			if tsResp.StatusCode == 200 {
				headOK = true
				tsCancel()
				break
			}
		}
		tsCancel()
		testedCount++
	}

	if !headOK {
		// 如果是 Master Playlist 且直接测试失败，可能需要递归解析子播放列表
		// 但对于可用性检查，当前实现已足够
		return false, 0, fmt.Errorf("TS分片不可访问")
	}

	// 深度检测
	if hc.deepCheck {
		if progressed, _ := hc.CheckProgressing(ctx, rawURL, userAgent); !progressed {
			return false, 0, fmt.Errorf("疑似循环/停滞的HLS播放列表")
		}
	}

	return true, time.Since(start).Milliseconds(), nil
}

// CheckProgressing 深度检测 HLS 是否在前进
func (hc *HLSChecker) CheckProgressing(ctx context.Context, playlistURL string, userAgent string) (bool, error) {
	type signature struct {
		mediaSeq string
		segments []string
		progTime string
	}

	parseSig := func(text string) signature {
		sig := signature{}
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				sig.mediaSeq = strings.TrimPrefix(line, "#EXT-X-MEDIA-SEQUENCE:")
			} else if strings.HasPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:") {
				sig.progTime = strings.TrimPrefix(line, "#EXT-X-PROGRAM-DATE-TIME:")
			} else if line != "" && !strings.HasPrefix(line, "#") {
				sig.segments = append(sig.segments, line)
			}
		}
		if len(sig.segments) > hc.segmentWindow {
			sig.segments = sig.segments[len(sig.segments)-hc.segmentWindow:]
		}
		return sig
	}

	var signatures []signature
	for i := 0; i < hc.loopChecks; i++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", playlistURL, nil)
		req.Header.Set("User-Agent", userAgent)
		resp, err := hc.client.Do(req)
		if err == nil {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return true, nil
			}
			signatures = append(signatures, parseSig(string(body)))
		} else {
			return true, nil
		}

		if i < hc.loopChecks-1 {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			case <-time.After(hc.loopInterval):
			}
		}
	}

	if len(signatures) < 2 {
		return true, nil
	}

	first := signatures[0]
	for _, sig := range signatures[1:] {
		if first.mediaSeq != "" && sig.mediaSeq != "" && sig.mediaSeq > first.mediaSeq {
			return true, nil
		}
		if len(sig.segments) != len(first.segments) {
			return true, nil
		}
		for i := range sig.segments {
			if sig.segments[i] != first.segments[i] {
				return true, nil
			}
		}
		if sig.progTime != "" && first.progTime != "" && sig.progTime != first.progTime {
			return true, nil
		}
	}

	return false, nil
}
