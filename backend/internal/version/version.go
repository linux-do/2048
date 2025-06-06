package version

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles static file versioning
type Manager struct {
	versions map[string]string
	mutex    sync.RWMutex
	baseDir  string
}

// NewManager creates a new version manager
func NewManager(staticDir string) *Manager {
	return &Manager{
		versions: make(map[string]string),
		baseDir:  staticDir,
	}
}

// GetVersion returns the version string for a static file
func (m *Manager) GetVersion(filePath string) string {
	m.mutex.RLock()
	version, exists := m.versions[filePath]
	m.mutex.RUnlock()

	if exists {
		return version
	}

	// Generate version if not cached
	version = m.generateVersion(filePath)
	
	m.mutex.Lock()
	m.versions[filePath] = version
	m.mutex.Unlock()

	return version
}

// generateVersion creates a version string based on file modification time and content hash
func (m *Manager) generateVersion(filePath string) string {
	fullPath := filepath.Join(m.baseDir, filePath)
	
	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		// If file doesn't exist, use timestamp as fallback
		return fmt.Sprintf("v%d", time.Now().Unix())
	}

	// Use modification time as primary version
	modTime := info.ModTime().Unix()

	// For additional uniqueness, calculate file hash
	hash := m.calculateFileHash(fullPath)
	if hash != "" {
		return fmt.Sprintf("v%d_%s", modTime, hash[:8])
	}

	return fmt.Sprintf("v%d", modTime)
}

// calculateFileHash calculates MD5 hash of file content
func (m *Manager) calculateFileHash(filePath string) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

// RefreshVersion forces regeneration of version for a specific file
func (m *Manager) RefreshVersion(filePath string) {
	m.mutex.Lock()
	delete(m.versions, filePath)
	m.mutex.Unlock()
}

// RefreshAll clears all cached versions
func (m *Manager) RefreshAll() {
	m.mutex.Lock()
	m.versions = make(map[string]string)
	m.mutex.Unlock()
}

// GetVersionedURL returns a URL with version parameter
func (m *Manager) GetVersionedURL(filePath string) string {
	version := m.GetVersion(filePath)
	return fmt.Sprintf("%s?%s", filePath, version)
}
