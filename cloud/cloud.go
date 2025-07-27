package cloud

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type CloudProvider struct {
	name    string
	secret  string
	api_key string
}

func New() *CloudProvider {
	return &CloudProvider{
		os.Getenv("CLOUDINARY_NAME"),
		os.Getenv("CLOUDINARY_API_SECRET"),
		os.Getenv("CLOUDINARY_API_KEY"),
	}
}

func (cloud *CloudProvider) GetCloudinarySignature(context *gin.Context) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	params := "timestamp=" + timestamp

	h := sha1.New()
	h.Write([]byte(params + cloud.secret))
	signature := hex.EncodeToString(h.Sum(nil))

	context.JSON(http.StatusOK, gin.H{
		"timestamp": timestamp,
		"signature": signature,
		"apiKey":    cloud.api_key,
		"cloudName": cloud.name,
	})
}

func UploadHandler(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files received"})
		return
	}

	var urls []string

	for _, file := range files {
		openedFile, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
			return
		}
		defer openedFile.Close()

		ext, reader, err := getFileExtension(openedFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to detect file type for: " + file.Filename})
			return
		}

		// Upload to Cloudinary
		url, err := uploadToCloudinary(reader, file.Filename, ext)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed for: " + file.Filename, "details": err.Error()})
			return
		}

		urls = append(urls, url)
	}

	c.JSON(http.StatusOK, gin.H{"url": urls})
}

func HandleFile(file *multipart.FileHeader) (string, error) {
	openedFile, err := file.Open()
	if err != nil {
		return "", err
	}
	defer openedFile.Close()

	ext, reader, err := getFileExtension(openedFile)
	if err != nil {
		return "", err
	}

	url, err := uploadToCloudinary(reader, file.Filename, ext)
	if err != nil {
		return "", err
	}

	return url, nil
}

func uploadToCloudinary(file io.Reader, filename, ext string) (string, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return "", fmt.Errorf("error loading .env: %w", err)
	}

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return "", fmt.Errorf("cloudinary init failed: %w", err)
	}

	publicID := filename[:len(filename)-len(filepath.Ext(filename))] // remove original extension

	uploadResp, err := cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
		PublicID: publicID,
	})
	if err != nil {
		return "", err
	}

	return uploadResp.SecureURL, nil
}

func getFileExtension(file io.Reader) (string, io.Reader, error) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "", nil, err
	}

	mimeType := http.DetectContentType(head[:n])
	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ".bin"
	if len(exts) > 0 {
		ext = exts[0]
	}

	fullReader := io.MultiReader(bytes.NewReader(head[:n]), file)
	return ext, fullReader, nil
}
