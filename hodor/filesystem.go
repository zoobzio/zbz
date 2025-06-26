package hodor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"zbz/zlog"
)

// FileSystemProvider provides file-based storage for hodor
type FileSystemProvider struct {
	basePath      string
	watcher       *fsnotify.Watcher
	subscriptions map[SubscriptionID]*fsSubscription
	mu            sync.RWMutex
	active        bool
}

type fsSubscription struct {
	id       SubscriptionID
	key      string
	callback ChangeCallback
}

// NewFileSystem creates a new file-based hodor contract
func NewFileSystem(basePath string) (*HodorContract, error) {
	// Ensure directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	
	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	
	// Add base path to watcher
	if err := watcher.Add(basePath); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch directory: %w", err)
	}
	
	provider := &FileSystemProvider{
		basePath:      basePath,
		watcher:       watcher,
		subscriptions: make(map[SubscriptionID]*fsSubscription),
		active:        true,
	}
	
	// Start watching for changes
	go provider.watchFiles()
	
	zlog.Info("Created file system storage", 
		zlog.String("path", basePath))
	
	return NewContract("filesystem", provider), nil
}

// NewFileSystemProvider creates a file system provider (for legacy registration)
func NewFileSystemProvider(config interface{}) (HodorProvider, error) {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("filesystem provider requires config map")
	}
	
	basePath, ok := configMap["path"].(string)
	if !ok {
		return nil, fmt.Errorf("filesystem provider requires 'path' config")
	}
	
	contract, err := NewFileSystem(basePath)
	if err != nil {
		return nil, err
	}
	
	return contract.Provider(), nil
}

// keyToPath converts a hodor key to a file path
func (fs *FileSystemProvider) keyToPath(key string) string {
	// Sanitize key to prevent directory traversal
	key = strings.ReplaceAll(key, "..", "")
	return filepath.Join(fs.basePath, key)
}

// pathToKey converts a file path back to a hodor key
func (fs *FileSystemProvider) pathToKey(path string) string {
	relPath, err := filepath.Rel(fs.basePath, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(relPath)
}

func (fs *FileSystemProvider) Get(key string) ([]byte, error) {
	path := fs.keyToPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

func (fs *FileSystemProvider) Set(key string, data []byte, ttl time.Duration) error {
	path := fs.keyToPath(key)
	
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Check if this is create or update
	operation := "create"
	if _, err := os.Stat(path); err == nil {
		operation = "update"
	}
	
	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	// Note: TTL not implemented for file system provider
	
	// Manually notify subscribers (fsnotify might not catch our own writes immediately)
	go fs.notifySubscribers(key, operation)
	
	return nil
}

func (fs *FileSystemProvider) Delete(key string) error {
	path := fs.keyToPath(key)
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Already deleted
	}
	
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	// Manually notify subscribers
	go fs.notifySubscribers(key, "delete")
	
	return nil
}

func (fs *FileSystemProvider) Exists(key string) (bool, error) {
	path := fs.keyToPath(key)
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (fs *FileSystemProvider) List(prefix string) ([]string, error) {
	var files []string
	
	startPath := fs.basePath
	if prefix != "" {
		startPath = fs.keyToPath(prefix)
	}
	
	err := filepath.WalkDir(startPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		if d.IsDir() {
			return nil // Skip directories
		}
		
		key := fs.pathToKey(path)
		if prefix == "" || strings.HasPrefix(key, prefix) {
			files = append(files, key)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	
	return files, nil
}

func (fs *FileSystemProvider) Stat(key string) (FileInfo, error) {
	path := fs.keyToPath(key)
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, fmt.Errorf("key '%s' not found", key)
		}
		return FileInfo{}, fmt.Errorf("failed to stat file: %w", err)
	}
	
	return FileInfo{
		Name:    key,
		Size:    stat.Size(),
		Mode:    uint32(stat.Mode()),
		ModTime: stat.ModTime(),
		IsDir:   stat.IsDir(),
	}, nil
}

func (fs *FileSystemProvider) GetProvider() string {
	return "filesystem"
}

func (fs *FileSystemProvider) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	fs.active = false
	
	if fs.watcher != nil {
		fs.watcher.Close()
	}
	
	fs.subscriptions = make(map[SubscriptionID]*fsSubscription)
	
	zlog.Info("Closed file system storage", 
		zlog.String("path", fs.basePath))
	
	return nil
}

func (fs *FileSystemProvider) Subscribe(key string, callback ChangeCallback) (SubscriptionID, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	id := SubscriptionID(uuid.New().String())
	
	subscription := &fsSubscription{
		id:       id,
		key:      key,
		callback: callback,
	}
	
	fs.subscriptions[id] = subscription
	
	zlog.Debug("Added file system subscription", 
		zlog.String("key", key),
		zlog.String("subscription_id", string(id)))
	
	return id, nil
}

func (fs *FileSystemProvider) Unsubscribe(id SubscriptionID) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	delete(fs.subscriptions, id)
	
	zlog.Debug("Removed file system subscription", 
		zlog.String("subscription_id", string(id)))
	
	return nil
}

// watchFiles monitors file system changes using fsnotify
func (fs *FileSystemProvider) watchFiles() {
	for {
		select {
		case event, ok := <-fs.watcher.Events:
			if !ok {
				return
			}
			
			if !fs.active {
				return
			}
			
			// Convert file path to key
			key := fs.pathToKey(event.Name)
			
			// Determine operation type
			var operation string
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				operation = "create"
			case event.Op&fsnotify.Write == fsnotify.Write:
				operation = "update"
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				operation = "delete"
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				operation = "delete" // Treat rename as delete for simplicity
			default:
				continue // Ignore other events
			}
			
			zlog.Debug("File system event", 
				zlog.String("key", key),
				zlog.String("operation", operation),
				zlog.String("path", event.Name))
			
			fs.notifySubscribers(key, operation)
			
		case err, ok := <-fs.watcher.Errors:
			if !ok {
				return
			}
			
			zlog.Warn("File system watcher error", zlog.Err(err))
		}
	}
}

// notifySubscribers notifies all relevant subscribers about changes
func (fs *FileSystemProvider) notifySubscribers(key, operation string) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	if !fs.active {
		return
	}
	
	event := ChangeEvent{
		Key:       key,
		Operation: operation,
		Timestamp: time.Now(),
	}
	
	// Get file size if it exists
	if operation != "delete" {
		if stat, err := os.Stat(fs.keyToPath(key)); err == nil {
			event.Size = stat.Size()
		}
	}
	
	// Notify relevant subscribers
	for _, sub := range fs.subscriptions {
		if sub.key == key {
			// Call callback in goroutine to avoid blocking
			go sub.callback(event)
		}
	}
}

// Register the filesystem provider
func init() {
	RegisterProvider("filesystem", NewFileSystemProvider)
}