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
	"strconv"
	"strings"
	"time"

	"example.com/models"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// func UploadHandler(c *gin.Context) {
// 	form, err := c.MultipartForm()
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
// 		return
// 	}

// 	files := form.File["files"]
// 	if len(files) == 0 {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "No files received"})
// 		return
// 	}

// 	var urls []string

// 	for _, file := range files {
// 		openedFile, err := file.Open()
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
// 			return
// 		}
// 		defer openedFile.Close()

// 		ext, reader, err := getFileExtension(openedFile)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to detect file type for: " + file.Filename})
// 			return
// 		}

// 		_, err = uploadToCloudinary(reader, file.Filename, ext)
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed for: " + file.Filename, "details": err.Error()})
// 			return
// 		}

// 	}

// 	c.JSON(http.StatusOK, gin.H{"url": urls})
// }

func HandleFile(file *multipart.FileHeader) (models.MediaItem, error) {
	openedFile, err := file.Open()
	if err != nil {
		return models.MediaItem{}, err
	}
	defer openedFile.Close()

	ext, reader, err := getFileExtension(openedFile)
	name := strings.TrimSuffix(file.Filename, ext)
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	if err != nil {
		return models.MediaItem{}, err
	}

	mediaItem, err := uploadToCloudinary(reader, file.Filename, name, mimeType)
	if err != nil {
		return models.MediaItem{}, err
	}

	return mediaItem, nil
}

func DeleteMedia(c *gin.Context, publicID string) {
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to initialize Cloudinary: " + err.Error()})
		return
	}

	resp, err := cld.Upload.Destroy(c, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete media: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Media deleted successfully",
		"result":  resp.Result,
	})
}

func uploadToCloudinary(file io.Reader, filename, name, mimeType string) (models.MediaItem, error) {
	err := godotenv.Load(".env")
	if err != nil {
		return models.MediaItem{}, fmt.Errorf("error loading .env: %w", err)
	}

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return models.MediaItem{}, fmt.Errorf("cloudinary init failed: %w", err)
	}

	publicID := uuid.New().String()

	// Check if the file is HEIC/HEIF and needs conversion
	uploadParams := uploader.UploadParams{
		PublicID: publicID,
	}

	// Convert HEIC/HEIF to JPG
	if strings.Contains(strings.ToLower(mimeType), "heic") ||
	   strings.Contains(strings.ToLower(mimeType), "heif") ||
	   strings.HasSuffix(strings.ToLower(filename), ".heic") ||
	   strings.HasSuffix(strings.ToLower(filename), ".heif") {
		uploadParams.Format = "jpg"
		mimeType = "image/jpeg"
	}

	uploadResp, err := cld.Upload.Upload(context.Background(), file, uploadParams)
	if err != nil {
		return models.MediaItem{}, err
	}

	return models.MediaItem{
		PublicID: publicID,
		URI:      uploadResp.SecureURL,
		MimeType: mimeType,
		FileName: name,
	}, nil
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
