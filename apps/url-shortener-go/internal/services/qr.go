package services

import (
	"bytes"
	"image/png"

	"github.com/skip2/go-qrcode"
)

// QRService handles QR code generation
type QRService struct{}

// NewQRService creates a new QR service
func NewQRService() *QRService {
	return &QRService{}
}

// GeneratePNG generates a QR code as PNG bytes
func (s *QRService) GeneratePNG(content string, size int) ([]byte, error) {
	// Validate size
	if size < 100 {
		size = 100
	}
	if size > 1000 {
		size = 1000
	}

	// Generate QR code
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	// Create buffer and encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, qr.Image(size)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GenerateSVG generates a QR code as SVG string
func (s *QRService) GenerateSVG(content string, size int) (string, error) {
	// Validate size
	if size < 100 {
		size = 100
	}
	if size > 1000 {
		size = 1000
	}

	// Generate QR code
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", err
	}

	// Generate SVG using the library's built-in method
	return qr.ToSmallString(false), nil
}
