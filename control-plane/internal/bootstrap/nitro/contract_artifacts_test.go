package nitro

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyChecksum_CorrectHash(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-artifact.zip")
	
	content := []byte("test artifact content for checksum verification")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)
	
	// Calculate the expected hash
	h := sha256.New()
	h.Write(content)
	expectedHash := fmt.Sprintf("sha256:%x", h.Sum(nil))
	
	// Temporarily add a test version to the checksums map
	testVersion := "test-version-1.0.0"
	originalChecksums := ArtifactChecksums
	ArtifactChecksums = map[string]string{
		testVersion: expectedHash,
	}
	defer func() { ArtifactChecksums = originalChecksums }()
	
	// Ensure verification is enabled
	originalSkip := SkipChecksumVerification
	SkipChecksumVerification = false
	defer func() { SkipChecksumVerification = originalSkip }()
	
	// Create downloader and verify
	downloader := NewContractArtifactDownloader(tmpDir)
	err = downloader.verifyChecksum(testFile, testVersion)
	assert.NoError(t, err, "correct checksum should pass verification")
}

func TestVerifyChecksum_WrongHash(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-artifact.zip")
	
	content := []byte("test artifact content for checksum verification")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)
	
	// Set a wrong expected hash
	testVersion := "test-version-wrong"
	originalChecksums := ArtifactChecksums
	ArtifactChecksums = map[string]string{
		testVersion: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}
	defer func() { ArtifactChecksums = originalChecksums }()
	
	// Ensure verification is enabled
	originalSkip := SkipChecksumVerification
	SkipChecksumVerification = false
	defer func() { SkipChecksumVerification = originalSkip }()
	
	// Create downloader and verify
	downloader := NewContractArtifactDownloader(tmpDir)
	err = downloader.verifyChecksum(testFile, testVersion)
	
	assert.Error(t, err, "wrong checksum should fail verification")
	assert.Contains(t, err.Error(), "checksum mismatch", "error should indicate checksum mismatch")
	assert.Contains(t, err.Error(), "SECURITY", "error should indicate security issue")
}

func TestVerifyChecksum_MissingVersion(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-artifact.zip")
	
	content := []byte("test artifact content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)
	
	// Use empty checksums map
	originalChecksums := ArtifactChecksums
	ArtifactChecksums = map[string]string{}
	defer func() { ArtifactChecksums = originalChecksums }()
	
	// Ensure verification is enabled
	originalSkip := SkipChecksumVerification
	SkipChecksumVerification = false
	defer func() { SkipChecksumVerification = originalSkip }()
	
	// Create downloader and verify
	downloader := NewContractArtifactDownloader(tmpDir)
	err = downloader.verifyChecksum(testFile, "unknown-version")
	
	assert.Error(t, err, "unknown version should fail verification")
	assert.Contains(t, err.Error(), "no checksum registered", "error should indicate missing checksum")
	assert.Contains(t, err.Error(), "SECURITY", "error should indicate security issue")
}

func TestVerifyChecksum_SkipInTestMode(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-artifact.zip")
	
	content := []byte("test artifact content")
	err := os.WriteFile(testFile, content, 0644)
	require.NoError(t, err)
	
	// Use empty checksums map (would normally fail)
	originalChecksums := ArtifactChecksums
	ArtifactChecksums = map[string]string{}
	defer func() { ArtifactChecksums = originalChecksums }()
	
	// Enable skip mode
	originalSkip := SkipChecksumVerification
	SkipChecksumVerification = true
	defer func() { SkipChecksumVerification = originalSkip }()
	
	// Create downloader and verify
	downloader := NewContractArtifactDownloader(tmpDir)
	err = downloader.verifyChecksum(testFile, "unknown-version")
	
	assert.NoError(t, err, "skip mode should bypass verification")
}

func TestVerifyChecksum_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Ensure verification is enabled
	originalSkip := SkipChecksumVerification
	SkipChecksumVerification = false
	defer func() { SkipChecksumVerification = originalSkip }()
	
	downloader := NewContractArtifactDownloader(tmpDir)
	err := downloader.verifyChecksum("/nonexistent/file.zip", "v1.0.0")
	
	assert.Error(t, err, "nonexistent file should fail")
	assert.Contains(t, err.Error(), "open file", "error should indicate file open issue")
}

func TestArtifactChecksums_HasCurrentVersion(t *testing.T) {
	// Verify that the current ArtifactVersion has a checksum entry
	_, ok := ArtifactChecksums[ArtifactVersion]
	assert.True(t, ok, "ArtifactChecksums should have an entry for current ArtifactVersion %s", ArtifactVersion)
}
