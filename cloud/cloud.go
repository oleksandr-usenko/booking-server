package cloud

import (
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCloudinarySignature(context *gin.Context) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	params := "timestamp=" + timestamp
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET") // Load from env

	h := sha1.New()
	h.Write([]byte(params + apiSecret))
	signature := hex.EncodeToString(h.Sum(nil))

	context.JSON(http.StatusOK, gin.H{
		"timestamp": timestamp,
		"signature": signature,
		"apiKey":    os.Getenv("CLOUDINARY_API_KEY"),
		"cloudName": os.Getenv("CLOUDINARY_NAME"),
	})
}
