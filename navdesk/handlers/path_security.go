package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
)

/**
 * 校验上传目录名称
 */
func isValidUploadDir(uploadDir string) bool {
	if uploadDir == "" {
		return false
	}

	for _, char := range uploadDir {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' {
			continue
		}
		return false
	}

	return true
}

/**
 * 解析基础目录内的相对路径
 */
func resolvePathWithinBase(basePath, relativePath string) (string, error) {
	normalizedPath := filepath.FromSlash(strings.TrimSpace(relativePath))
	if normalizedPath == "" || filepath.IsAbs(normalizedPath) {
		return "", fmt.Errorf("路径必须是非空相对路径")
	}

	cleanPath := filepath.Clean(normalizedPath)
	parentPrefix := ".." + string(filepath.Separator)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, parentPrefix) {
		return "", fmt.Errorf("路径超出允许目录")
	}

	baseAbsolutePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", err
	}

	targetAbsolutePath := filepath.Join(baseAbsolutePath, cleanPath)
	relativeToBase, err := filepath.Rel(baseAbsolutePath, targetAbsolutePath)
	if err != nil {
		return "", err
	}
	if relativeToBase == "." || relativeToBase == ".." || strings.HasPrefix(relativeToBase, parentPrefix) {
		return "", fmt.Errorf("路径超出允许目录")
	}

	return targetAbsolutePath, nil
}

/**
 * 解析上传目录路径
 */
func resolveUploadDirectory(uploadsPath, uploadDir string) (string, error) {
	if !isValidUploadDir(uploadDir) {
		return "", fmt.Errorf("上传目录名称不合法")
	}
	return resolvePathWithinBase(uploadsPath, uploadDir)
}

/**
 * 解析本地上传文件路径
 */
func resolveUploadedFilePath(uploadsPath, fileURL string) (string, error) {
	const uploadsPrefix = "/uploads/"
	if !strings.HasPrefix(fileURL, uploadsPrefix) {
		return "", fmt.Errorf("文件地址不属于上传目录")
	}

	return resolvePathWithinBase(uploadsPath, strings.TrimPrefix(fileURL, uploadsPrefix))
}
