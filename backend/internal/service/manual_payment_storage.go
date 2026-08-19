package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	_ "golang.org/x/image/webp"
)

const (
	manualQRMaxUploadBytes    int64 = 1 << 20
	manualProofMaxUploadBytes int64 = 5 << 20
	manualImageMaxPixels            = 25_000_000
	manualQRPayloadMaxBytes         = 4096
	manualPrivateStorageEnv         = "PAYMENT_PRIVATE_STORAGE_DIR"
)

const (
	ManualQRMaxUploadBytes    = manualQRMaxUploadBytes
	ManualProofMaxUploadBytes = manualProofMaxUploadBytes
)

type manualStoredFile struct {
	StorageKey string
	MIMEType   string
	Size       int64
	SHA256     string
	QRPayload  string
}

type manualPaymentStorage struct {
	root string
}

func newManualPaymentStorage() (*manualPaymentStorage, error) {
	root := strings.TrimSpace(os.Getenv(manualPrivateStorageEnv))
	if root == "" {
		root = filepath.Join("data", "payment-private")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve manual payment storage: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create manual payment storage: %w", err)
	}
	return &manualPaymentStorage{root: abs}, nil
}

func (s *manualPaymentStorage) storeQR(reader io.Reader, channel string) (*manualStoredFile, error) {
	img, format, err := decodeManualPaymentImage(reader, manualQRMaxUploadBytes)
	if err != nil {
		return nil, err
	}
	payload, err := decodeManualQRCode(img)
	if err != nil {
		return nil, err
	}
	if err := validateManualQRPayload(channel, payload); err != nil {
		return nil, err
	}
	_ = format
	return s.encodeAndStore("qr", img, "png", payload)
}

func (s *manualPaymentStorage) storeProof(reader io.Reader) (*manualStoredFile, error) {
	img, format, err := decodeManualPaymentImage(reader, manualProofMaxUploadBytes)
	if err != nil {
		return nil, err
	}
	outputFormat := "jpeg"
	if format == "png" {
		outputFormat = "png"
	}
	return s.encodeAndStore("proof", img, outputFormat, "")
}

func decodeManualPaymentImage(reader io.Reader, maxBytes int64) (image.Image, string, error) {
	if reader == nil {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "image file is required")
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "failed to read image")
	}
	if int64(len(data)) == 0 {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "image file is empty")
	}
	if int64(len(data)) > maxBytes {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_IMAGE_TOO_LARGE", fmt.Sprintf("image exceeds %d bytes", maxBytes))
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || !manualPaymentImageFormatAllowed(format) {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "only PNG, JPEG, and WebP images are accepted")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > manualImageMaxPixels {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "image dimensions are invalid or too large")
	}
	img, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decodedFormat != format {
		return nil, "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_IMAGE", "image data is malformed")
	}
	return img, format, nil
}

func manualPaymentImageFormatAllowed(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "png", "jpeg", "webp":
		return true
	default:
		return false
	}
}

func decodeManualQRCode(img image.Image) (string, error) {
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_QR_UNREADABLE", "unable to read QR code")
	}
	result, err := qrcode.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil || result == nil {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_QR_UNREADABLE", "no readable QR code was found")
	}
	payload := strings.TrimSpace(result.GetText())
	if payload == "" || len(payload) > manualQRPayloadMaxBytes {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_QR_INVALID", "QR code payload is empty or too long")
	}
	return payload, nil
}

func validateManualQRPayload(channel, payload string) error {
	channel = NormalizeVisibleMethod(channel)
	u, err := url.Parse(strings.TrimSpace(payload))
	if err != nil || u.Scheme == "" {
		return infraerrors.BadRequest("MANUAL_PAYMENT_QR_INVALID", "QR code does not contain a supported payment link")
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	valid := false
	switch channel {
	case payment.TypeAlipay:
		valid = (scheme == "https" && host == "qr.alipay.com") ||
			((scheme == "alipays" || scheme == "alipay") && strings.EqualFold(u.Host, "platformapi"))
	case payment.TypeWxpay:
		valid = scheme == "wxp" ||
			(scheme == "weixin" && strings.HasPrefix(strings.ToLower(u.Host+u.Path), "wxpay/")) ||
			(scheme == "https" && (host == "payapp.weixin.qq.com" || host == "wx.tenpay.com"))
	}
	if !valid {
		return infraerrors.BadRequest("MANUAL_PAYMENT_QR_CHANNEL_MISMATCH", "QR code is not a supported link for the selected channel")
	}
	return nil
}

func (s *manualPaymentStorage) encodeAndStore(kind string, img image.Image, format, payload string) (*manualStoredFile, error) {
	if s == nil || s.root == "" {
		return nil, fmt.Errorf("manual payment storage is not configured")
	}
	var encoded bytes.Buffer
	ext := ".png"
	mimeType := "image/png"
	if format == "jpeg" {
		ext = ".jpg"
		mimeType = "image/jpeg"
		if err := jpeg.Encode(&encoded, flattenImageOnWhite(img), &jpeg.Options{Quality: 88}); err != nil {
			return nil, fmt.Errorf("encode payment image: %w", err)
		}
	} else if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&encoded, img); err != nil {
		return nil, fmt.Errorf("encode payment image: %w", err)
	}

	storageKey := filepath.ToSlash(filepath.Join(kind, uuid.NewString()+ext))
	fullPath, err := s.resolve(storageKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		return nil, fmt.Errorf("create private payment directory: %w", err)
	}
	tmpPath := fullPath + ".tmp-" + uuid.NewString()
	if err := os.WriteFile(tmpPath, encoded.Bytes(), 0o600); err != nil {
		return nil, fmt.Errorf("write private payment image: %w", err)
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("commit private payment image: %w", err)
	}
	_ = os.Chmod(fullPath, 0o600)
	hash := sha256.Sum256(encoded.Bytes())
	return &manualStoredFile{
		StorageKey: storageKey,
		MIMEType:   mimeType,
		Size:       int64(encoded.Len()),
		SHA256:     hex.EncodeToString(hash[:]),
		QRPayload:  payload,
	}, nil
}

func flattenImageOnWhite(src image.Image) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	draw.Draw(dst, bounds, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Over)
	return dst
}

func (s *manualPaymentStorage) read(storageKey string) ([]byte, error) {
	fullPath, err := s.resolve(storageKey)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, infraerrors.NotFound("MANUAL_PAYMENT_IMAGE_NOT_FOUND", "payment image is no longer available")
		}
		return nil, fmt.Errorf("read private payment image: %w", err)
	}
	return data, nil
}

func (s *manualPaymentStorage) remove(storageKey string) error {
	fullPath, err := s.resolve(storageKey)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove private payment image: %w", err)
	}
	return nil
}

func (s *manualPaymentStorage) resolve(storageKey string) (string, error) {
	if s == nil || s.root == "" {
		return "", fmt.Errorf("manual payment storage is not configured")
	}
	storageKey = strings.TrimSpace(filepath.ToSlash(storageKey))
	if storageKey == "" || strings.HasPrefix(storageKey, "/") || strings.Contains(storageKey, "..") {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_STORAGE_KEY", "invalid private payment storage key")
	}
	fullPath := filepath.Clean(filepath.Join(s.root, filepath.FromSlash(storageKey)))
	rel, err := filepath.Rel(s.root, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", infraerrors.BadRequest("MANUAL_PAYMENT_INVALID_STORAGE_KEY", "invalid private payment storage key")
	}
	return fullPath, nil
}
