package util

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"runtime"
	"runtime/debug"
	"time"

	log "github.com/sirupsen/logrus"
	qrcode "github.com/skip2/go-qrcode"
)

func CheckAndPanic(e error) {
	if e != nil {
		panic(e)
	}
}

func CheckAndLog(e error) {
	if e != nil {
		log.Error(e)
	}
}

func ConvertPaymentHash(paymentHash string) []int {
	rHashByteSlice, err := hex.DecodeString(paymentHash)
	CheckAndPanic(err)

	rHashIntSlice := []int{}
	for _, rHashByte := range rHashByteSlice {
		rHashIntSlice = append(rHashIntSlice, int(rHashByte))
	}

	return rHashIntSlice
}

func Random_hex() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	CheckAndPanic(err)

	return hex.EncodeToString(b)
}

func Max(a, b int) int {
	var max int
	if a > b {
		max = a
	} else {
		max = b
	}
	return max
}

func QrPngBase64Encode(data string) (encoded string) {
	var data_qr_png []byte
	data_qr_png, err := qrcode.Encode(data, qrcode.Medium, 256)
	if err != nil {
		log.Warn("qrcode error: ", err.Error())
	}

	// https://stackoverflow.com/questions/2807251/can-i-embed-a-png-image-into-an-html-page
	encoded = base64.StdEncoding.EncodeToString(data_qr_png)

	return encoded
}

// free memory to the OS regularly so that docker container memory remains reasonable
func FreeMemory() {
	for {
		time.Sleep(1 * time.Minute)
		runtime.GC()
		debug.FreeOSMemory()
	}
}
