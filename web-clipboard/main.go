package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "clipboard_session"
const sessionTTL = 30 * 24 * time.Hour
const maxLoginFailures = 5
const loginFailureWindow = 15 * time.Minute
const loginLockoutDuration = 15 * time.Minute
const defaultConfigPath = "clipboard-config.json"
const defaultAddr = ":8080"
const defaultDataPath = "clipboard.json"
const bytesPerMB int64 = 1024 * 1024
const defaultMaxMB int64 = 1

//go:embed static/index.html
var indexHTML string

type config struct {
	configPath   string
	configLoaded bool
	addr         string
	dataPath     string
	creds        credentials
	maxBytes     int64
}

type userConfig struct {
	Addr      string    `json:"addr"`
	DataPath  string    `json:"dataPath"`
	MaxMB     int64     `json:"maxMB"`
	MaxBytes  int64     `json:"maxBytes,omitempty"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"createdAt"`
}

type credentials struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"createdAt"`
}

type clipboardState struct {
	Text      string    `json:"text"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type fileStore struct {
	mu    sync.Mutex
	path  string
	state clipboardState
}

type loginAttempt struct {
	failures    int
	lastFailed  time.Time
	lockedUntil time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
}

type server struct {
	store        *fileStore
	username     string
	password     string
	maxBytes     int64
	loginLimiter *loginLimiter
}

// 启动 Web 剪贴板服务。
func main() {
	cfg, generated, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.maxBytes <= 0 {
		log.Fatal("配置项 maxMB 必须大于 0")
	}

	store, err := newFileStore(cfg.dataPath)
	if err != nil {
		log.Fatalf("加载数据文件失败: %v", err)
	}

	app := &server{
		store:        store,
		username:     cfg.creds.Username,
		password:     cfg.creds.Password,
		maxBytes:     cfg.maxBytes,
		loginLimiter: newLoginLimiter(),
	}

	httpServer := &http.Server{
		Addr:              cfg.addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Web 剪贴板服务监听地址: %s", cfg.addr)
	log.Printf("用户配置文件: %s", cfg.configPath)
	if !cfg.configLoaded {
		log.Printf("未找到用户配置文件，已生成默认配置")
	}
	log.Printf("数据文件: %s", cfg.dataPath)
	auditLog(nil, "service_start", auditField("addr", cfg.addr), auditField("config", cfg.configPath), auditField("data", cfg.dataPath))
	if generated {
		log.Printf("已生成账号密码并写入用户配置文件")
		log.Printf("用户名: %s", cfg.creds.Username)
		log.Printf("密码: %s", cfg.creds.Password)
	}
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// 加载运行配置并准备账号密码。
func loadConfig() (config, bool, error) {
	cfg := config{
		addr:     defaultAddr,
		dataPath: defaultDataPath,
		maxBytes: defaultMaxMB * bytesPerMB,
	}
	flag.StringVar(&cfg.configPath, "config", defaultConfigPath, "用户配置文件")
	flag.Parse()

	cfg.configPath = strings.TrimSpace(cfg.configPath)
	if cfg.configPath == "" {
		cfg.configPath = defaultConfigPath
	}

	loaded, generated, err := loadUserConfig(cfg.configPath, &cfg)
	if err != nil {
		return cfg, false, err
	}
	cfg.configLoaded = loaded

	cfg.dataPath = strings.TrimSpace(cfg.dataPath)
	cfg.addr = strings.TrimSpace(cfg.addr)
	cfg.creds.Username = strings.TrimSpace(cfg.creds.Username)
	if cfg.addr == "" {
		cfg.addr = defaultAddr
	}
	if cfg.dataPath == "" {
		cfg.dataPath = defaultDataPath
	}
	return cfg, generated, nil
}

// 读取用户配置文件并补齐缺失的账号密码。
func loadUserConfig(path string, cfg *config) (bool, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		creds, err := newCredentials()
		if err != nil {
			return false, false, fmt.Errorf("生成账号密码失败: %w", err)
		}
		cfg.creds = creds
		if err := saveUserConfig(path, cfg); err != nil {
			return false, false, fmt.Errorf("保存用户配置文件失败: %w", err)
		}
		return false, true, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("读取用户配置文件失败: %w", err)
	}

	var fileConfig userConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return false, false, fmt.Errorf("解析用户配置文件失败: %w", err)
	}
	if strings.TrimSpace(fileConfig.Addr) != "" {
		cfg.addr = fileConfig.Addr
	}
	if strings.TrimSpace(fileConfig.DataPath) != "" {
		cfg.dataPath = fileConfig.DataPath
	}
	if fileConfig.MaxMB > 0 {
		cfg.maxBytes = fileConfig.MaxMB * bytesPerMB
	} else if fileConfig.MaxBytes > 0 {
		cfg.maxBytes = fileConfig.MaxBytes
	}
	cfg.creds = credentials{
		Username:  fileConfig.Username,
		Password:  fileConfig.Password,
		CreatedAt: fileConfig.CreatedAt,
	}
	if strings.TrimSpace(cfg.creds.Username) == "" || cfg.creds.Password == "" {
		creds, err := newCredentials()
		if err != nil {
			return false, false, fmt.Errorf("生成账号密码失败: %w", err)
		}
		cfg.creds = creds
		if err := saveUserConfig(path, cfg); err != nil {
			return true, false, fmt.Errorf("保存用户配置文件失败: %w", err)
		}
		return true, true, nil
	}
	return true, false, nil
}

// 生成新的登录账号密码。
func newCredentials() (credentials, error) {
	username, err := randomString(6)
	if err != nil {
		return credentials{}, err
	}
	password, err := randomString(24)
	if err != nil {
		return credentials{}, err
	}
	return credentials{
		Username:  "user-" + username,
		Password:  password,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// 生成指定长度的随机字符串。
func randomString(byteCount int) (string, error) {
	value := make([]byte, byteCount)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

// 保存用户配置文件。
func saveUserConfig(path string, cfg *config) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	fileConfig := userConfig{
		Addr:      cfg.addr,
		DataPath:  cfg.dataPath,
		MaxMB:     cfg.maxBytes / bytesPerMB,
		Username:  cfg.creds.Username,
		Password:  cfg.creds.Password,
		CreatedAt: cfg.creds.CreatedAt,
	}
	data, err := json.MarshalIndent(fileConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// 创建并加载文件存储。
func newFileStore(path string) (*fileStore, error) {
	store := &fileStore{path: path}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// 加载已保存的剪贴板内容。
func (s *fileStore) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.state = clipboardState{UpdatedAt: time.Now().UTC()}
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		s.state = clipboardState{UpdatedAt: time.Now().UTC()}
		return nil
	}

	var state clipboardState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	s.state = state
	return nil
}

// 读取当前剪贴板状态。
func (s *fileStore) get() clipboardState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// 更新当前剪贴板文本。
func (s *fileStore) update(text string) (clipboardState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := clipboardState{
		Text:      text,
		UpdatedAt: time.Now().UTC(),
	}
	if err := s.save(next); err != nil {
		return s.state, err
	}
	s.state = next
	return next, nil
}

// 保存剪贴板状态到文件。
func (s *fileStore) save(state clipboardState) error {
	dir := filepath.Dir(s.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// 注册 HTTP 路由。
func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/clipboard", s.handleClipboard)
	mux.HandleFunc("/healthz", s.handleHealthz)
	return securityHeaders(mux)
}

// 返回 Web 页面。
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

// 处理浏览器登录和退出登录。
func (s *server) handleSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if retryAfter, ok := s.loginLimiter.blocked(auditRemoteAddr(r), time.Now().UTC()); ok {
			writeRateLimited(w, retryAfter)
			auditLog(r, "login_blocked", auditField("reason", "登录已锁定"), auditField("retry_after", retryAfterSeconds(retryAfter)))
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 4096)

		var payload struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			s.handleLoginFailure(w, r, "", "请求格式不正确", http.StatusBadRequest, "请求内容格式不正确")
			return
		}
		if !s.validCredentials(payload.Username, payload.Password) {
			s.handleLoginFailure(w, r, strings.TrimSpace(payload.Username), "用户名或密码错误", http.StatusUnauthorized, "用户名或密码错误")
			return
		}
		s.loginLimiter.reset(auditRemoteAddr(r))

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    s.newSessionCookie(time.Now().UTC()),
			Path:     "/",
			MaxAge:   int(sessionTTL.Seconds()),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})
		auditLog(r, "login_success", auditField("username", strings.TrimSpace(payload.Username)))
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case http.MethodDelete:
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   r.TLS != nil,
		})
		auditLog(r, "logout")
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, http.MethodPost, http.MethodDelete)
	}
}

// 处理登录失败并记录限速状态。
func (s *server) handleLoginFailure(w http.ResponseWriter, r *http.Request, username, reason string, status int, message string) {
	failures, retryAfter, locked := s.loginLimiter.addFailure(auditRemoteAddr(r), time.Now().UTC())
	fields := []string{auditField("reason", reason), auditField("failures", strconv.Itoa(failures))}
	if username != "" {
		fields = append(fields, auditField("username", username))
	}
	if locked {
		fields = append(fields, auditField("retry_after", retryAfterSeconds(retryAfter)))
		auditLog(r, "login_blocked", fields...)
		writeRateLimited(w, retryAfter)
		return
	}

	auditLog(r, "login_failed", fields...)
	writeError(w, status, message)
}

// 处理剪贴板读取和保存。
func (s *server) handleClipboard(w http.ResponseWriter, r *http.Request) {
	if !s.requireAuth(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		state := s.store.get()
		auditLog(r, "clipboard_read", auditField("username", s.username))
		writeJSON(w, http.StatusOK, state)
	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, s.maxBytes+4096)

		var payload struct {
			Text string `json:"text"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			auditLog(r, "clipboard_save_failed", auditField("username", s.username), auditField("reason", "请求格式不正确"))
			writeError(w, http.StatusBadRequest, "请求内容格式不正确")
			return
		}
		textBytes := len([]byte(payload.Text))
		if int64(textBytes) > s.maxBytes {
			auditLog(r, "clipboard_save_failed", auditField("username", s.username), auditField("reason", "文本内容过大"), auditField("bytes", strconv.Itoa(textBytes)))
			writeError(w, http.StatusRequestEntityTooLarge, "文本内容过大")
			return
		}

		state, err := s.store.update(payload.Text)
		if err != nil {
			auditLog(r, "clipboard_save_failed", auditField("username", s.username), auditField("reason", "保存失败"), auditField("bytes", strconv.Itoa(textBytes)))
			writeError(w, http.StatusInternalServerError, "保存剪贴板失败")
			return
		}
		auditLog(r, "clipboard_saved", auditField("username", s.username), auditField("bytes", strconv.Itoa(textBytes)))
		writeJSON(w, http.StatusOK, state)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPut)
	}
}

// 返回服务健康状态。
func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// 校验当前请求是否已认证。
func (s *server) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if username, password, ok := r.BasicAuth(); ok && s.validCredentials(username, password) {
		return true
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil && s.validSessionCookie(cookie.Value, time.Now().UTC()) {
		return true
	}

	auditLog(r, "auth_failed", auditField("auth", authMethod(r)))
	writeError(w, http.StatusUnauthorized, "未登录或登录已过期")
	return false
}

// 校验用户名和密码。
func (s *server) validCredentials(username, password string) bool {
	username = strings.TrimSpace(username)
	if len(username) != len(s.username) || len(password) != len(s.password) {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.password)) == 1
	return userOK && passOK
}

// 生成登录会话 Cookie 值。
func (s *server) newSessionCookie(now time.Time) string {
	username := base64.RawURLEncoding.EncodeToString([]byte(s.username))
	message := fmt.Sprintf("%s.%d", username, now.Add(sessionTTL).Unix())
	return message + "." + s.signSession(message)
}

// 校验登录会话 Cookie。
func (s *server) validSessionCookie(value string, now time.Time) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}

	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || now.Unix() > expiresAt {
		return false
	}

	message := parts[0] + "." + parts[1]
	if !constantTimeStringEqual(parts[2], s.signSession(message)) {
		return false
	}

	username, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	return constantTimeStringEqual(string(username), s.username)
}

// 生成会话签名。
func (s *server) signSession(message string) string {
	mac := hmac.New(sha256.New, []byte(s.password))
	_, _ = mac.Write([]byte("web-clipboard-session:"))
	_, _ = mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// 以固定时间方式比较字符串。
func constantTimeStringEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

// 写入 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// 写入错误响应。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// 返回请求方法不允许响应。
func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "请求方法不允许")
}

// 创建登录限速器。
func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: make(map[string]loginAttempt)}
}

// 判断来源 IP 是否已被锁定。
func (l *loginLimiter) blocked(ip string, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, ok := l.attempts[ip]
	if !ok || attempt.lockedUntil.IsZero() {
		return 0, false
	}
	if now.Before(attempt.lockedUntil) {
		return attempt.lockedUntil.Sub(now), true
	}
	delete(l.attempts, ip)
	return 0, false
}

// 记录来源 IP 的登录失败。
func (l *loginLimiter) addFailure(ip string, now time.Time) (int, time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt := l.attempts[ip]
	if !attempt.lockedUntil.IsZero() && !now.Before(attempt.lockedUntil) {
		attempt = loginAttempt{}
	}
	if !attempt.lastFailed.IsZero() && now.Sub(attempt.lastFailed) > loginFailureWindow {
		attempt = loginAttempt{}
	}
	attempt.failures++
	attempt.lastFailed = now
	if attempt.failures >= maxLoginFailures {
		attempt.lockedUntil = now.Add(loginLockoutDuration)
		l.attempts[ip] = attempt
		return attempt.failures, loginLockoutDuration, true
	}
	l.attempts[ip] = attempt
	return attempt.failures, 0, false
}

// 清除来源 IP 的登录失败记录。
func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// 写入登录限速响应。
func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
	writeError(w, http.StatusTooManyRequests, "登录尝试过多，请稍后再试")
}

// 返回 Retry-After 秒数。
func retryAfterSeconds(retryAfter time.Duration) string {
	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// 写入安全审计日志。
func auditLog(r *http.Request, event string, fields ...string) {
	parts := []string{"SECURITY_AUDIT", auditField("event", event), auditField("message", auditMessage(event))}
	if r != nil {
		parts = append(parts, auditField("remote", auditRemoteAddr(r)), auditField("method", r.Method), auditField("path", r.URL.Path))
	}
	parts = append(parts, fields...)
	log.Print(strings.Join(parts, " "))
}

// 返回安全审计事件中文说明。
func auditMessage(event string) string {
	switch event {
	case "service_start":
		return "服务启动"
	case "login_success":
		return "登录成功"
	case "login_failed":
		return "登录失败"
	case "login_blocked":
		return "登录被限速"
	case "logout":
		return "退出登录"
	case "auth_failed":
		return "认证失败"
	case "clipboard_read":
		return "读取剪贴板"
	case "clipboard_saved":
		return "保存剪贴板成功"
	case "clipboard_save_failed":
		return "保存剪贴板失败"
	default:
		return "安全审计事件"
	}
}

// 格式化安全审计字段。
func auditField(key, value string) string {
	return key + "=" + strconv.Quote(value)
}

// 提取安全审计来源地址。
func auditRemoteAddr(r *http.Request) string {
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if value := strings.TrimSpace(parts[0]); value != "" {
			return value
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// 判断请求携带的认证方式。
func authMethod(r *http.Request) string {
	methods := make([]string, 0, 2)
	if _, _, ok := r.BasicAuth(); ok {
		methods = append(methods, "basic")
	}
	if _, err := r.Cookie(sessionCookieName); err == nil {
		methods = append(methods, "cookie")
	}
	if len(methods) == 0 {
		return "none"
	}
	return strings.Join(methods, "+")
}

// 为响应添加安全相关请求头。
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
