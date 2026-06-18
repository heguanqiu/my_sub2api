package service

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// PluginObjectStore 复用备份的对象存储能力
type PluginObjectStore = BackupObjectStore

// PluginStoreProvider 返回配置好的对象存储客户端
type PluginStoreProvider interface {
	Store(ctx context.Context) (PluginObjectStore, error)
}

type pluginStoreProvider struct {
	settingRepo  SettingRepository
	encryptor    SecretEncryptor
	storeFactory BackupObjectStoreFactory
}

func NewPluginStoreProvider(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	storeFactory BackupObjectStoreFactory,
) PluginStoreProvider {
	return &pluginStoreProvider{
		settingRepo:  settingRepo,
		encryptor:    encryptor,
		storeFactory: storeFactory,
	}
}

func (p *pluginStoreProvider) Store(ctx context.Context) (PluginObjectStore, error) {
	raw, err := p.settingRepo.GetValue(ctx, settingKeyBackupS3Config)
	if err != nil || raw == "" {
		return nil, ErrPluginStorageNotConfigured
	}
	var cfg BackupS3Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, ErrBackupS3ConfigCorrupt
	}
	if cfg.SecretAccessKey != "" {
		if decrypted, derr := p.encryptor.Decrypt(cfg.SecretAccessKey); derr != nil {
			logger.LegacyPrintf("service.plugin", "[Plugin] S3 SecretAccessKey 解密失败（可能是旧的未加密数据）: %v", derr)
		} else {
			cfg.SecretAccessKey = decrypted
		}
	}
	if !cfg.IsConfigured() {
		return nil, ErrPluginStorageNotConfigured
	}
	return p.storeFactory(ctx, &cfg)
}
