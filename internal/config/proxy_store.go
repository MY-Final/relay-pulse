package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"monitor/internal/apikey"
)

// ProxyProfilesFileName 是管理员代理配置文件名。
const ProxyProfilesFileName = "proxies.yaml"

// ProxyProfile 是一条可复用的管理员代理配置。URL 只在服务端内存中保留，
// 持久化时写入 url_encrypted，API 响应也不会返回明文 URL。
type ProxyProfile struct {
	ID        string `yaml:"id" json:"id"`
	Name      string `yaml:"name" json:"name"`
	URL       string `yaml:"-" json:"-"`
	CreatedAt string `yaml:"created_at" json:"created_at"`
	UpdatedAt string `yaml:"updated_at" json:"updated_at"`
}

type proxyProfileDisk struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	URL          string `yaml:"url,omitempty"` // 兼容早期手工配置，新增写入一律加密
	URLEncrypted string `yaml:"url_encrypted,omitempty"`
	CreatedAt    string `yaml:"created_at"`
	UpdatedAt    string `yaml:"updated_at"`
}

type proxyProfilesDisk struct {
	Proxies []proxyProfileDisk `yaml:"proxies"`
}

// ProxyProfileStore 管理 config/proxies.yaml。所有写操作串行化并原子落盘。
type ProxyProfileStore struct {
	path      string
	keyCipher *apikey.KeyCipher
	mu        sync.Mutex
}

// NewProxyProfileStore 创建代理配置存储。
func NewProxyProfileStore(path string, keyCipher *apikey.KeyCipher) *ProxyProfileStore {
	return &ProxyProfileStore{path: path, keyCipher: keyCipher}
}

// Path 返回代理配置文件路径。
func (s *ProxyProfileStore) Path() string { return s.path }

func (s *ProxyProfileStore) loadLocked() ([]ProxyProfile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取代理配置失败: %w", err)
	}

	var disk proxyProfilesDisk
	if err := yaml.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("解析代理配置失败: %w", err)
	}

	profiles := make([]ProxyProfile, 0, len(disk.Proxies))
	seen := make(map[string]struct{}, len(disk.Proxies))
	for i, item := range disk.Proxies {
		id := strings.TrimSpace(item.ID)
		name := strings.TrimSpace(item.Name)
		if id == "" || name == "" {
			return nil, fmt.Errorf("代理配置[%d] 的 id 和 name 不能为空", i)
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("代理配置 id 重复: %s", id)
		}
		seen[id] = struct{}{}

		proxyURL := strings.TrimSpace(item.URL)
		if item.URLEncrypted != "" {
			if s.keyCipher == nil {
				return nil, fmt.Errorf("代理配置 %s 已加密，但管理员加密密钥不可用", id)
			}
			proxyURL, err = s.keyCipher.Decrypt(item.URLEncrypted)
			if err != nil {
				return nil, fmt.Errorf("解密代理配置 %s 失败: %w", id, err)
			}
			proxyURL = strings.TrimSpace(proxyURL)
		}
		if proxyURL == "" {
			return nil, fmt.Errorf("代理配置 %s 的地址不能为空", id)
		}
		if err := validateProxyURL(proxyURL); err != nil {
			return nil, fmt.Errorf("代理配置 %s 无效: %w", id, err)
		}

		profiles = append(profiles, ProxyProfile{
			ID:        id,
			Name:      name,
			URL:       proxyURL,
			CreatedAt: strings.TrimSpace(item.CreatedAt),
			UpdatedAt: strings.TrimSpace(item.UpdatedAt),
		})
	}
	return profiles, nil
}

// List 返回全部代理配置。返回值包含 URL，仅供服务端内部使用，API 层必须脱敏后输出。
func (s *ProxyProfileStore) List() ([]ProxyProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

// Get 按 ID 获取代理配置。
func (s *ProxyProfileStore) Get(id string) (*ProxyProfile, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("代理配置 id 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		if profiles[i].ID == id {
			profile := profiles[i]
			return &profile, nil
		}
	}
	return nil, nil
}

func newProxyProfileID(existing map[string]struct{}) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("生成代理配置 id 失败: %w", err)
		}
		id := "proxy-" + hex.EncodeToString(buf)
		if _, ok := existing[id]; !ok {
			return id, nil
		}
	}
	return "", fmt.Errorf("生成代理配置 id 冲突")
}

func normalizeProxyProfileInput(name, proxyURL string) (string, string, error) {
	name = strings.TrimSpace(name)
	proxyURL = strings.TrimSpace(proxyURL)
	if name == "" {
		return "", "", fmt.Errorf("代理名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", "", fmt.Errorf("代理名称不能超过 100 个字符")
	}
	if proxyURL == "" {
		return "", "", fmt.Errorf("代理地址不能为空")
	}
	if err := validateProxyURL(proxyURL); err != nil {
		return "", "", err
	}
	return name, proxyURL, nil
}

func (s *ProxyProfileStore) saveLocked(profiles []ProxyProfile) error {
	disk := proxyProfilesDisk{Proxies: make([]proxyProfileDisk, 0, len(profiles))}
	for _, profile := range profiles {
		if s.keyCipher == nil {
			return fmt.Errorf("保存代理配置需要管理员加密密钥")
		}
		if strings.TrimSpace(profile.URL) == "" {
			return fmt.Errorf("代理配置 %s 的地址不能为空", profile.ID)
		}
		encrypted, err := s.keyCipher.Encrypt(profile.URL)
		if err != nil {
			return fmt.Errorf("加密代理配置 %s 失败: %w", profile.ID, err)
		}
		disk.Proxies = append(disk.Proxies, proxyProfileDisk{
			ID:           profile.ID,
			Name:         profile.Name,
			URLEncrypted: encrypted,
			CreatedAt:    profile.CreatedAt,
			UpdatedAt:    profile.UpdatedAt,
		})
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("创建代理配置目录失败: %w", err)
	}
	return AtomicWriteYAML(s.path, disk)
}

// Create 创建代理配置。
func (s *ProxyProfileStore) Create(name, proxyURL string) (*ProxyProfile, error) {
	name, proxyURL, err := normalizeProxyProfileInput(name, proxyURL)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	existing := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		existing[profile.ID] = struct{}{}
	}
	id, err := newProxyProfileID(existing)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	profile := ProxyProfile{ID: id, Name: name, URL: proxyURL, CreatedAt: now, UpdatedAt: now}
	profiles = append(profiles, profile)
	if err := s.saveLocked(profiles); err != nil {
		return nil, err
	}
	return &profile, nil
}

// Update 更新代理配置。
func (s *ProxyProfileStore) Update(id, name, proxyURL string) (*ProxyProfile, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	proxyURL = strings.TrimSpace(proxyURL)
	if name == "" {
		return nil, fmt.Errorf("代理名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return nil, fmt.Errorf("代理名称不能超过 100 个字符")
	}
	if proxyURL != "" {
		if err := validateProxyURL(proxyURL); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		if profiles[i].ID != id {
			continue
		}
		profiles[i].Name = name
		if proxyURL != "" {
			profiles[i].URL = proxyURL
		}
		profiles[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := s.saveLocked(profiles); err != nil {
			return nil, err
		}
		profile := profiles[i]
		return &profile, nil
	}
	return nil, fmt.Errorf("代理配置不存在: %s", id)
}

// Delete 删除代理配置。调用方应先确保没有监测通道引用它。
func (s *ProxyProfileStore) Delete(id string) error {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	profiles, err := s.loadLocked()
	if err != nil {
		return err
	}
	filtered := profiles[:0]
	found := false
	for _, profile := range profiles {
		if profile.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, profile)
	}
	if !found {
		return fmt.Errorf("代理配置不存在: %s", id)
	}
	return s.saveLocked(filtered)
}

// LoadProxyProfileURLs 读取代理配置，供配置加载器把 proxy_profile 解析为运行时 Proxy。
func LoadProxyProfileURLs(configDir, encryptionKey string) (map[string]string, error) {
	var cipher *apikey.KeyCipher
	if strings.TrimSpace(encryptionKey) != "" {
		var err error
		cipher, err = apikey.NewKeyCipher(encryptionKey)
		if err != nil {
			return nil, err
		}
	}
	store := NewProxyProfileStore(filepath.Join(configDir, ProxyProfilesFileName), cipher)
	profiles, err := store.List()
	if err != nil {
		return nil, err
	}
	urls := make(map[string]string, len(profiles))
	for _, profile := range profiles {
		urls[profile.ID] = profile.URL
	}
	return urls, nil
}

// ResolveMonitorProxyProfiles 将通道引用的代理配置 ID 解析为运行时代理 URL。
// 旧的 proxy 字段仍可直接使用；proxy_profile 存在时以代理配置为准。
func ResolveMonitorProxyProfiles(monitors []ServiceConfig, urls map[string]string) error {
	for i := range monitors {
		id := strings.TrimSpace(monitors[i].ProxyProfile)
		if id == "" {
			continue
		}
		proxyURL, ok := urls[id]
		if !ok {
			return fmt.Errorf("monitor[%d] 引用的代理配置不存在: %s", i, id)
		}
		monitors[i].Proxy = proxyURL
	}
	return nil
}
