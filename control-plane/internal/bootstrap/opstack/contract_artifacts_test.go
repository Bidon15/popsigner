package opstack

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyChecksum(t *testing.T) {
	t.Run("valid checksum passes verification", func(t *testing.T) {
		// Create a temporary file with known content
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-artifact.tzst")
		testContent := []byte("test artifact content for checksum verification")

		err := os.WriteFile(testFile, testContent, 0644)
		require.NoError(t, err)

		// Calculate the expected hash
		expectedHash := fmt.Sprintf("%x", sha256.Sum256(testContent))

		// Temporarily add a test version to the checksums map
		ArtifactChecksums["test-v1"] = expectedHash
		defer delete(ArtifactChecksums, "test-v1")

		// Create downloader and verify
		downloader := NewContractArtifactDownloader(tmpDir)
		err = downloader.verifyChecksum(testFile, "test-v1")
		assert.NoError(t, err)
	})

	t.Run("invalid checksum fails verification", func(t *testing.T) {
		// Create a temporary file with known content
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-artifact.tzst")
		testContent := []byte("test artifact content")

		err := os.WriteFile(testFile, testContent, 0644)
		require.NoError(t, err)

		// Set a wrong checksum
		ArtifactChecksums["test-v2"] = "0000000000000000000000000000000000000000000000000000000000000000"
		defer delete(ArtifactChecksums, "test-v2")

		// Create downloader and verify - should fail
		downloader := NewContractArtifactDownloader(tmpDir)
		err = downloader.verifyChecksum(testFile, "test-v2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("empty checksum skips verification with warning", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-artifact.tzst")
		err := os.WriteFile(testFile, []byte("any content"), 0644)
		require.NoError(t, err)

		// Set empty checksum (verification should be skipped)
		ArtifactChecksums["test-v3"] = ""
		defer delete(ArtifactChecksums, "test-v3")

		// Create downloader and verify - should pass (skipped)
		downloader := NewContractArtifactDownloader(tmpDir)
		err = downloader.verifyChecksum(testFile, "test-v3")
		assert.NoError(t, err)
	})

	t.Run("missing version fails verification", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test-artifact.tzst")
		err := os.WriteFile(testFile, []byte("content"), 0644)
		require.NoError(t, err)

		// Don't add any checksum for this version
		downloader := NewContractArtifactDownloader(tmpDir)
		err = downloader.verifyChecksum(testFile, "nonexistent-version")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no checksum configured")
	})

	t.Run("file not found returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		ArtifactChecksums["test-v4"] = "somehash"
		defer delete(ArtifactChecksums, "test-v4")

		downloader := NewContractArtifactDownloader(tmpDir)
		err := downloader.verifyChecksum(filepath.Join(tmpDir, "nonexistent.tzst"), "test-v4")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "open file for checksum")
	})
}

func TestArtifactChecksums(t *testing.T) {
	t.Run("current version has checksum entry", func(t *testing.T) {
		// Verify that the current ArtifactVersion has an entry in the checksums map
		_, ok := ArtifactChecksums[ArtifactVersion]
		assert.True(t, ok, "ArtifactVersion %s should have an entry in ArtifactChecksums", ArtifactVersion)
	})
}
