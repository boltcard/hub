package crypto

import (
	"crypto/aes"
	"crypto/cipher"
)

// decrypt p with aes_dec
func Aes_decrypt(key_sdm_file_read []byte, ba_p []byte) ([]byte, error) {

	dec_p := make([]byte, 16)
	iv := make([]byte, 16)
	c1, err := aes.NewCipher(key_sdm_file_read)
	if err != nil {
		return dec_p, err
	}
	mode := cipher.NewCBCDecrypter(c1, iv)
	mode.CryptBlocks(dec_p, ba_p)

	return dec_p, nil
}
