package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Target 定义支持的浏览器/平台组合
type Target int

const (
	ChromeWin Target = iota
	ChromeAndroid
	SafariIPhone
	SafariMac
)

// Version 表示浏览器版本信息
type Version struct {
	Major int
	Full  string
}

// Result 表示生成的 User-Agent 结果
type Result struct {
	UserAgent      string
	BrowserVersion Version
}

// Options 定义生成选项
type Options struct {
	MaxMajor     int
	MajorDelta   int
	MobileVendor string
}

// Option 函数式选项类型
type Option func(*Options)

// WithMaxMajor 设置最大主版本号
func WithMaxMajor(v int) Option {
	return func(o *Options) { o.MaxMajor = v }
}

// WithMajorDelta 设置版本号范围差值
func WithMajorDelta(v int) Option {
	return func(o *Options) { o.MajorDelta = v }
}

// WithMobileVendor 设置移动设备型号
func WithMobileVendor(v string) Option {
	return func(o *Options) { o.MobileVendor = v }
}

// 版本号范围常量
var (
	chromeVersionRange = struct {
		MajorMin, MajorMax int
		PatchMin, PatchMax int
		BuildMin, BuildMax int
	}{136, 138, 6834, 7204, 85, 101}

	safariVersionRange = struct {
		MajorMin, MajorMax int
		MinorMin, MinorMax int
		PatchMin, PatchMax int
	}{614, 632, 1, 36, 1, 15}
)

// Android 设备型号列表
var mobileVendors = []string{
	"SM-T510", "SM-T295", "SM-T515", "SM-T860", "SM-T720", "SM-T595", "SM-T290", "SM-T865", "SM-T835",
	"SM-T725", "SM-P610", "SM-T590", "SM-P615", "TV BOX", "SM-T830", "Lenovo TB-X505X", "SM-T500",
	"Lenovo TB-X505F", "Lenovo TB-X606F", "SM-P205", "SM-T505", "MRX-W09", "Lenovo YT-X705F",
	"Lenovo TB-X505L", "MRX-AL09", "SCM-W09", "Lenovo TB-X606X", "P20HD_EEA", "SM-A105M", "iPlay_20",
	"Lenovo TB-X606V", "H96 Max RK3318", "TVBOX", "SM-T387V", "Lenovo YT-X705L", "Lenovo TB-X306X",
	"Lenovo TB-X306F", "SM-T870", "Redmi Note 8 Pro", "Tab8", "SM-T970", "SM-A205G", "Lenovo TB-X605FC",
	"Lenovo TB-J606F", "e-tab 20", "ADT1061", "SM-T307U", "100003562", "MBOX", "Lenovo TB-X605LC",
	"M40_EEA", "M2003J15SC", "100003561", "X109", "Redmi Note 8", "Lenovo TB-8705F", "A860", "SM-A107M",
	"Redmi Note 7", "BAH3-W09", "BAH3-L09", "TX6s", "SM-T507", "P20HD_ROW", "Magnet_G30", "SM-T875",
	"SM-T387W", "MI PAD 4", "Lenovo YT-X705X", "Lenovo TB-X606FA", "SM-P200", "SM-A207M", "M2004J19C",
	"X104-EEA", "SM-T837V", "SM-A307GT", "AGS3-W09", "SM-T505N", "SM-A105F", "Magnet_G50", "A850", "8092",
	"100015685-A", "X88pro10.q2.0.6330.d4", "SM-T975", "SM-G973F", "J5",
}

// Windows NT 版本
const windowsNTVersion = "10.0"

// Windows 架构
const windowsArch = "Win64"

// macOS 版本选项
var macOSVersions = []string{
	"10_15_7", "10_14_6", "10_13_6", "11_0", "11_1", "11_2", "11_3", "11_4", "11_5", "11_6",
	"12_0", "12_1", "12_2", "12_3", "12_4", "12_5", "12_6",
	"13_0", "13_1", "13_2", "13_3", "13_4", "13_5", "13_6",
	"14_0", "14_1", "14_2", "14_3", "14_4", "14_5", "14_6",
}

// Android 版本选项
var androidVersions = []string{"9", "10", "11", "12", "13", "14", "14", "14", "14"}

// iOS 版本选项
var iOSVersions = []string{
	"16_0", "16_1", "16_2", "16_3", "16_4", "16_5",
	"17_0", "17_1", "17_2", "17_3", "17_4", "17_5",
}

// Safari 浏览器版本选项
var safariBrowserVersions = []string{"15", "16", "17", "17", "17"}

// fromRange 生成 [min, max] 闭区间随机整数
func fromRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}

// randChoice 从切片中随机选择一个元素
func randChoice[T any](s []T) T {
	return s[rand.Intn(len(s))]
}

// chromeVersion 生成 Chrome 版本号
func chromeVersion(maxMajor, majorDelta int) Version {
	minMajor, maxMaj := chromeVersionRange.MajorMin, chromeVersionRange.MajorMax
	if maxMajor > 0 {
		maxMaj = maxMajor
		minMajor = max(maxMajor-majorDelta, 0)
	}

	major := fromRange(minMajor, maxMaj)
	patch := fromRange(chromeVersionRange.PatchMin, chromeVersionRange.PatchMax)
	build := fromRange(chromeVersionRange.BuildMin, chromeVersionRange.BuildMax)

	return Version{
		Major: major,
		Full:  fmt.Sprintf("%d.0.%d.%d", major, patch, build),
	}
}

// safariVersion 生成 Safari 版本号
func safariVersion(maxMajor, majorDelta int) Version {
	minMajor, maxMaj := safariVersionRange.MajorMin, safariVersionRange.MajorMax
	if maxMajor > 0 {
		maxMaj = maxMajor
		minMajor = max(maxMajor-majorDelta, 0)
	}

	major := fromRange(minMajor, maxMaj)
	minor := fromRange(safariVersionRange.MinorMin, safariVersionRange.MinorMax)

	full := fmt.Sprintf("%d.%d", major, minor)
	if rand.Float64() < 0.3 {
		patch := fromRange(safariVersionRange.PatchMin, safariVersionRange.PatchMax)
		full = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	}

	return Version{Major: major, Full: full}
}

// generateChromeWindows 生成 Chrome Windows UA
func generateChromeWindows(ver Version) string {
	return fmt.Sprintf(
		"Mozilla/5.0 (Windows NT %s; %s; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36",
		windowsNTVersion, windowsArch, ver.Full,
	)
}

// generateChromeAndroid 生成 Chrome Android UA
func generateChromeAndroid(ver Version, vendor string) string {
	androidVer := randChoice(androidVersions)
	return fmt.Sprintf(
		"Mozilla/5.0 (Linux; Android %s; %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36",
		androidVer, vendor, ver.Full,
	)
}

// generateSafariIPhone 生成 Safari iPhone UA
func generateSafariIPhone(ver Version) string {
	iosVer := randChoice(iOSVersions)

	// 从 iOS 版本提取 Version（如 17_5 -> 17.5）
	browserVer := strings.ReplaceAll(iosVer, "_", ".")

	// 生成随机 Mobile 标识
	mobileID := fmt.Sprintf("%c%c%c%c%c%c",
		'A'+rand.Intn(26), '0'+rand.Intn(10), '0'+rand.Intn(10),
		'A'+rand.Intn(26), '0'+rand.Intn(10), 'A'+rand.Intn(26),
	)

	return fmt.Sprintf(
		"Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Mobile/%s Safari/%s",
		iosVer, browserVer, mobileID, ver.Full,
	)
}

// generateSafariMac 生成 Safari macOS UA
func generateSafariMac(ver Version) string {
	macVer := randChoice(macOSVersions)
	browserVer := randChoice(safariBrowserVersions)
	minorVer := fromRange(0, 7)

	return fmt.Sprintf(
		"Mozilla/5.0 (Macintosh; Intel Mac OS X %s) AppleWebKit/%s (KHTML, like Gecko) Version/%s.%d Safari/%s",
		macVer, ver.Full, browserVer, minorVer, ver.Full,
	)
}

// Generate 生成指定目标的 User-Agent
func Generate(target Target, opts ...Option) Result {
	options := &Options{MajorDelta: 2}
	for _, opt := range opts {
		opt(options)
	}

	switch target {
	case ChromeWin:
		ver := chromeVersion(options.MaxMajor, options.MajorDelta)
		return Result{
			UserAgent:      generateChromeWindows(ver),
			BrowserVersion: ver,
		}

	case ChromeAndroid:
		ver := chromeVersion(options.MaxMajor, options.MajorDelta)
		vendor := options.MobileVendor
		if vendor == "" {
			vendor = randChoice(mobileVendors)
		}
		return Result{
			UserAgent:      generateChromeAndroid(ver, vendor),
			BrowserVersion: ver,
		}

	case SafariIPhone:
		ver := safariVersion(options.MaxMajor, options.MajorDelta)
		return Result{
			UserAgent:      generateSafariIPhone(ver),
			BrowserVersion: ver,
		}

	case SafariMac:
		ver := safariVersion(options.MaxMajor, options.MajorDelta)
		return Result{
			UserAgent:      generateSafariMac(ver),
			BrowserVersion: ver,
		}

	default:
		return Result{}
	}
}

func main() {
	// 读取并规范化 URL_PREFIX
	prefix := "/" + strings.Trim(os.Getenv("URL_PREFIX"), "/")
	http.HandleFunc("GET "+prefix, handleUA)
	if prefix != "/" {
		http.HandleFunc("GET "+prefix+"/", handleUA)
	}

	port := ":8080"
	fmt.Printf("Server starting on %s, route: %s\n", port, prefix)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Response API 响应结构
type Response struct {
	Code string   `json:"code"`
	UA   []string `json:"ua"`
}

// handleUA 处理 /ua 请求
func handleUA(w http.ResponseWriter, r *http.Request) {
	// 调试日志：获取真实 IP
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	fmt.Printf("%s %s %s\n", time.Now().Format("2006-01-02 15:04:05"), ip, r.URL.RawQuery)

	query := r.URL.Query()

	// 解析数量参数
	count := 1
	if countStr := query.Get("count"); countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
			count = n
		}
	}

	// 限制最大数量
	if count > 100 {
		count = 100
	}

	// 解析设备类型参数
	typeMap := map[string]Target{
		"win":     ChromeWin,
		"android": ChromeAndroid,
		"ios":     SafariIPhone,
		"mac":     SafariMac,
	}

	var targets []Target
	if typeStr := query.Get("type"); typeStr != "" {
		for t := range strings.SplitSeq(typeStr, "@") {
			if target, ok := typeMap[strings.TrimSpace(t)]; ok {
				targets = append(targets, target)
			}
		}
	}

	// 未指定时使用所有设备
	if len(targets) == 0 {
		targets = []Target{ChromeWin, ChromeAndroid, SafariIPhone, SafariMac}
	}

	// 生成 UA
	uas := make([]string, count)
	for i := 0; i < count; i++ {
		target := targets[rand.Intn(len(targets))]
		uas[i] = Generate(target).UserAgent
	}

	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	resp := Response{Code: "200", UA: uas}
	json.NewEncoder(w).Encode(resp)
}
