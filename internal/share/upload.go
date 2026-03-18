package share

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const uploadTimeout = 30 * time.Second

// UploadResult contains the download URL after upload.
type UploadResult struct {
	URL      string
	Provider string
}

// Upload uploads data to a temporary file hosting service.
// Tries tmpfiles.org first, falls back to file.io.
func Upload(ctx context.Context, data []byte, filename string) (*UploadResult, error) {
	result, err := uploadTmpfiles(ctx, data, filename)
	if err == nil {
		return result, nil
	}
	// Fallback to file.io
	result2, err2 := uploadFileIO(ctx, data, filename)
	if err2 != nil {
		return nil, fmt.Errorf("tmpfiles: %w; file.io: %w", err, err2)
	}
	return result2, nil
}

// Download downloads data from a URL.
func Download(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func uploadTmpfiles(ctx context.Context, data []byte, filename string) (*UploadResult, error) {
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	body, contentType, err := buildMultipart("file", filename, data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://tmpfiles.org/api/v1/upload", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tmpfiles: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("tmpfiles: decode: %w", err)
	}
	if result.Status != "success" || result.Data.URL == "" {
		return nil, errors.New("tmpfiles: unexpected response")
	}

	// Convert view URL to direct download URL
	// https://tmpfiles.org/1234/file.ext → https://tmpfiles.org/dl/1234/file.ext
	dlURL := strings.Replace(result.Data.URL, "tmpfiles.org/", "tmpfiles.org/dl/", 1)

	return &UploadResult{URL: dlURL, Provider: "tmpfiles.org"}, nil
}

func uploadFileIO(ctx context.Context, data []byte, filename string) (*UploadResult, error) {
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	body, contentType, err := buildMultipart("file", filename, data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://file.io", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file.io: HTTP %d", resp.StatusCode)
	}

	var result struct {
		Success bool   `json:"success"`
		Link    string `json:"link"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("file.io: decode: %w", err)
	}
	if !result.Success || result.Link == "" {
		return nil, errors.New("file.io: upload failed")
	}

	return &UploadResult{URL: result.Link, Provider: "file.io"}, nil
}

func buildMultipart(field, filename string, data []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, filename)
	if err != nil {
		return nil, "", err
	}
	if _, err := fw.Write(data); err != nil {
		return nil, "", err
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}
