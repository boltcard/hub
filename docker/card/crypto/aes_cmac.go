package crypto

import (
	"crypto/aes"
	"crypto/subtle"

	"github.com/aead/cmac"
)

func Aes_cmac(key_sdm_file_read_mac []byte, sv2 []byte, ba_c []byte) (bool, error) {

	c2, err := aes.NewCipher(key_sdm_file_read_mac)
	if err != nil {
		return false, err
	}
	ks, err := cmac.Sum(sv2, c2, 16)
	if err != nil {
		return false, err
	}
	c3, err := aes.NewCipher(ks)
	if err != nil {
		return false, err
	}
	cm, err := cmac.Sum([]byte{}, c3, 16)
	if err != nil {
		return false, err
	}
	ct := make([]byte, 8)
	ct[0] = cm[1]
	ct[1] = cm[3]
	ct[2] = cm[5]
	ct[3] = cm[7]
	ct[4] = cm[9]
	ct[5] = cm[11]
	ct[6] = cm[13]
	ct[7] = cm[15]

	if subtle.ConstantTimeCompare(ct, ba_c) != 1 {
		return false, nil
	}

	return true, nil
}
