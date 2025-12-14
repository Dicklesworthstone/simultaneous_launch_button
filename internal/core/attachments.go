// Package core implements attachment handling for SLB requests.
package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // Register JPEG format
	_ "image/png"  // Register PNG format
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Dicklesworthstone/slb/internal/db"
)

// AttachmentConfig holds configuration for attachment handling.
type AttachmentConfig struct {
	// MaxFileSize is the maximum size for file attachments (default 1MB).
	MaxFileSize int64
	// MaxOutputSize is the maximum output size for context commands (default 100KB).
	MaxOutputSize int64
	// MaxImageSize is the maximum dimension for images (default 4096x4096).
	MaxImageSize int
	// AllowedFileTypes restricts file types (empty means all allowed).
	AllowedFileTypes []string
}

// DefaultAttachmentConfig returns default configuration.
func DefaultAttachmentConfig() AttachmentConfig {
	return AttachmentConfig{
		MaxFileSize:      1024 * 1024,      // 1MB
		MaxOutputSize:    100 * 1024,       // 100KB
		MaxImageSize:     4096,             // 4096px
		AllowedFileTypes: []string{},      // Allow all
	}
}

// AttachmentError represents an attachment processing error.
type AttachmentError struct {
	Type    db.AttachmentType
	Path    string
	Message string
}

func (e *AttachmentError) Error() string {
	return fmt.Sprintf("attachment error (%s): %s", e.Type, e.Message)
}

// LoadAttachmentFromFile reads a file and creates an attachment.
func LoadAttachmentFromFile(path string, config *AttachmentConfig) (*db.Attachment, error) {
	if config == nil {
		cfg := DefaultAttachmentConfig()
		config = &cfg
	}

	// Resolve path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("resolving path: %v", err),
		}
	}

	// Check file exists and get info
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("stat: %v", err),
		}
	}

	// Check size
	if config.MaxFileSize > 0 && info.Size() > config.MaxFileSize {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("file too large: %d bytes (max %d)", info.Size(), config.MaxFileSize),
		}
	}

	// Read content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("reading file: %v", err),
		}
	}

	// Detect if this is an image
	attachType := db.AttachmentTypeFile
	if isImageFile(absPath) {
		attachType = db.AttachmentTypeScreenshot
	} else if isDiffFile(absPath) || isDiffContent(content) {
		attachType = db.AttachmentTypeGitDiff
	}

	// For images, encode as base64 data URI
	var contentStr string
	if attachType == db.AttachmentTypeScreenshot {
		mimeType := detectImageMimeType(absPath)
		contentStr = fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content))
	} else {
		contentStr = string(content)
	}

	return &db.Attachment{
		Type:    attachType,
		Content: contentStr,
		Metadata: map[string]any{
			"source":   absPath,
			"filename": filepath.Base(absPath),
			"size":     info.Size(),
		},
	}, nil
}

// LoadScreenshot loads an image file as a screenshot attachment.
func LoadScreenshot(path string, config *AttachmentConfig) (*db.Attachment, error) {
	if config == nil {
		cfg := DefaultAttachmentConfig()
		config = &cfg
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeScreenshot,
			Path:    path,
			Message: fmt.Sprintf("resolving path: %v", err),
		}
	}

	// Verify it's an image
	if !isImageFile(absPath) {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeScreenshot,
			Path:    path,
			Message: "file is not a recognized image format",
		}
	}

	// Open and validate image dimensions
	f, err := os.Open(absPath)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeScreenshot,
			Path:    path,
			Message: fmt.Sprintf("opening file: %v", err),
		}
	}
	defer f.Close()

	imgConfig, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeScreenshot,
			Path:    path,
			Message: fmt.Sprintf("decoding image: %v", err),
		}
	}

	if config.MaxImageSize > 0 {
		if imgConfig.Width > config.MaxImageSize || imgConfig.Height > config.MaxImageSize {
			return nil, &AttachmentError{
				Type:    db.AttachmentTypeScreenshot,
				Path:    path,
				Message: fmt.Sprintf("image too large: %dx%d (max %d)", imgConfig.Width, imgConfig.Height, config.MaxImageSize),
			}
		}
	}

	// Read and encode
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeScreenshot,
			Path:    path,
			Message: fmt.Sprintf("reading file: %v", err),
		}
	}

	mimeType := detectImageMimeType(absPath)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content))

	return &db.Attachment{
		Type:    db.AttachmentTypeScreenshot,
		Content: dataURI,
		Metadata: map[string]any{
			"source":      absPath,
			"filename":    filepath.Base(absPath),
			"width":       imgConfig.Width,
			"height":      imgConfig.Height,
			"description": "",
		},
	}, nil
}

// RunContextCommand executes a command and captures output as an attachment.
func RunContextCommand(command string, config *AttachmentConfig) (*db.Attachment, error) {
	if config == nil {
		cfg := DefaultAttachmentConfig()
		config = &cfg
	}

	// Run the command
	cmd := exec.Command("sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine output
	var output strings.Builder
	output.WriteString(stdout.String())
	if stderr.Len() > 0 {
		if output.Len() > 0 {
			output.WriteString("\n--- stderr ---\n")
		}
		output.WriteString(stderr.String())
	}

	// Check output size
	outputStr := output.String()
	if config.MaxOutputSize > 0 && int64(len(outputStr)) > config.MaxOutputSize {
		// Truncate with notice
		outputStr = outputStr[:config.MaxOutputSize] + "\n... [truncated]"
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return &db.Attachment{
		Type:    db.AttachmentTypeContext,
		Content: outputStr,
		Metadata: map[string]any{
			"source":    command,
			"exit_code": exitCode,
		},
	}, nil
}

// CreateLogExcerpt creates a log excerpt attachment from a file.
func CreateLogExcerpt(path string, startLine, endLine int, config *AttachmentConfig) (*db.Attachment, error) {
	if config == nil {
		cfg := DefaultAttachmentConfig()
		config = &cfg
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("resolving path: %v", err),
		}
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, &AttachmentError{
			Type:    db.AttachmentTypeFile,
			Path:    path,
			Message: fmt.Sprintf("reading file: %v", err),
		}
	}

	lines := strings.Split(string(content), "\n")

	// Adjust line numbers (1-indexed to 0-indexed)
	if startLine < 1 {
		startLine = 1
	}
	if endLine < 1 || endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		startLine = endLine
	}

	// Extract lines
	excerpt := strings.Join(lines[startLine-1:endLine], "\n")

	return &db.Attachment{
		Type:    db.AttachmentTypeFile, // Log excerpts are a type of file attachment
		Content: excerpt,
		Metadata: map[string]any{
			"file":        absPath,
			"lines":       fmt.Sprintf("%d-%d", startLine, endLine),
			"total_lines": len(lines),
			"type":        "log_excerpt",
		},
	}, nil
}

// CreateDiffAttachment creates a diff attachment from git or a file.
func CreateDiffAttachment(diffContent string, ref string) *db.Attachment {
	return &db.Attachment{
		Type:    db.AttachmentTypeGitDiff,
		Content: diffContent,
		Metadata: map[string]any{
			"ref": ref,
		},
	}
}

// Helper functions

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".bmp":  true,
		".webp": true,
	}
	return imageExts[ext]
}

func detectImageMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeTypes := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func isDiffFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".diff" || ext == ".patch"
}

func isDiffContent(content []byte) bool {
	s := string(content)
	// Look for diff markers
	return strings.HasPrefix(s, "diff ") ||
		strings.HasPrefix(s, "--- ") ||
		strings.HasPrefix(s, "@@")
}
