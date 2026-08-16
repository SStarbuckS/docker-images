package imap

import (
	"fmt"
	"log"
	"strings"
	"time"

	idle "github.com/emersion/go-imap-idle"
	"github.com/emersion/go-imap/client"
)

// IdleClient IDLE客户端封装
type IdleClient struct {
	client       *client.Client
	idleClient   *idle.Client
	accountName  string
	idleTimeout  time.Duration
	supportsIDLE bool
}

// IdleResult 表示一次 IDLE 周期的结果。
type IdleResult struct {
	HasUpdate bool
	Err       error
}

// NewIdleClient 创建IDLE客户端
func NewIdleClient(c *client.Client, accountName string, idleTimeoutMinutes int) *IdleClient {
	return &IdleClient{
		client:      c,
		idleClient:  idle.NewClient(c),
		accountName: accountName,
		idleTimeout: time.Duration(idleTimeoutMinutes) * time.Minute,
	}
}

// isConnectionError 检查是否是连接错误
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "EOF") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "reset by peer") ||
		strings.Contains(errMsg, "wsasend") ||
		strings.Contains(errMsg, "aborted")
}

// CheckIDLESupport 检查服务器是否支持IDLE扩展
func (ic *IdleClient) CheckIDLESupport() (bool, error) {
	supported, err := ic.idleClient.SupportIdle()
	if err != nil {
		ic.supportsIDLE = false
		return false, fmt.Errorf("检查 IDLE 支持失败: %w", err)
	}
	ic.supportsIDLE = supported
	return supported, nil
}

// MonitorWithIDLE 使用IDLE监控邮箱（一次性模式）
func (ic *IdleClient) MonitorWithIDLE(folder string) <-chan IdleResult {
	updateCh := make(chan IdleResult)

	go func() {
		defer close(updateCh)

		// 选择文件夹并启动IDLE
		if _, err := ic.client.Select(folder, false); err != nil {
			updateCh <- IdleResult{Err: fmt.Errorf("选择文件夹 %s 失败: %w", folder, err)}
			return
		}

		hasUpdate, err := ic.runIDLE()

		if err != nil {
			// 发生错误
			if isConnectionError(err) {
				log.Printf("[%s] 连接断开，重新建立连接", ic.accountName)
			} else {
				log.Printf("[%s] IDLE 错误: %v", ic.accountName, err)
			}
			updateCh <- IdleResult{Err: err}
		} else if hasUpdate {
			// 收到新邮件通知
			updateCh <- IdleResult{HasUpdate: true}
		} else {
			// 正常超时
			log.Printf("[%s] IDLE 超时 (%v)，重新进入 IDLE", ic.accountName, ic.idleTimeout)
			updateCh <- IdleResult{}
		}
	}()

	return updateCh
}

// runIDLE 执行IDLE命令，返回(是否有更新, 错误)
func (ic *IdleClient) runIDLE() (bool, error) {
	// 创建停止通道
	idleStop := make(chan struct{})
	var idleStopClosed bool
	closeIdleStop := func() {
		if !idleStopClosed {
			close(idleStop)
			idleStopClosed = true
		}
	}

	// 创建更新通道
	updates := make(chan client.Update, 10)
	ic.client.Updates = updates
	defer func() {
		ic.client.Updates = nil
	}()

	// 启动IDLE协程
	idleDone := make(chan error, 1)
	go func() {
		idleDone <- ic.idleClient.Idle(idleStop)
	}()

	// 设置超时（使用配置的超时时间）
	timeout := time.NewTimer(ic.idleTimeout)
	defer timeout.Stop()

	// 等待更新
	for {
		select {
		case <-timeout.C:
			// 超时，停止IDLE
			closeIdleStop()
			<-idleDone
			return false, nil // 无更新，无错误

		case update := <-updates:
			// 收到更新
			closeIdleStop()
			<-idleDone

			if update != nil {
				return true, nil // 有更新，无错误
			}
			return false, nil // 无更新，无错误

		case err := <-idleDone:
			// IDLE结束
			closeIdleStop()
			if err != nil {
				return false, fmt.Errorf("IDLE错误: %w", err)
			}
			return false, nil // IDLE正常结束，无更新
		}
	}
}
