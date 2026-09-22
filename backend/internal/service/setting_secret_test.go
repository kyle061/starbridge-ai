//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRevealSecretDecryptsStoredS3Credentials(t *testing.T) {
	ctx := context.Background()
	svc, repo, _ := newImageStorageFixture(t, config.ImageStorageConfig{})
	raw, err := json.Marshal(BackupS3Config{Bucket: "backups", SecretAccessKey: "enc:backup-secret"})
	require.NoError(t, err)
	require.NoError(t, repo.Set(ctx, settingKeyBackupS3Config, string(raw)))
	value, err := svc.backup.GetStoredS3Secret(ctx)
	require.NoError(t, err)
	require.Equal(t, "backup-secret", value)
	public, err := svc.backup.GetS3Config(ctx)
	require.NoError(t, err)
	require.Empty(t, public.SecretAccessKey)
	for _, reuse := range []bool{false, true} {
		raw, err = json.Marshal(ImageStorageSettings{ReuseBackupS3: reuse, SecretAccessKey: "enc:image-secret"})
		require.NoError(t, err)
		require.NoError(t, repo.Set(ctx, settingKeyImageStorageConfig, string(raw)))
		value, err = svc.GetStoredS3Secret(ctx)
		require.NoError(t, err)
		if reuse {
			require.Equal(t, "backup-secret", value)
		} else {
			require.Equal(t, "image-secret", value)
		}
	}
}
