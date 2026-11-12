package lddata

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type AesCrypt struct {
	passphrase string
}

func NewAesCrypt(passphrase string) *AesCrypt {
	result := AesCrypt{passphrase: passphrase}
	return &result
}

func (c *AesCrypt) Decrypt(cryptoJSText string) (string, error) {
	if len(cryptoJSText) < 0 {
		return "", fmt.Errorf("invalid CryptoJS text: length=%d", len(cryptoJSText))
	}
	// 1. Decode base64 string
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoJSText)
	if err != nil {
		return "", err
	}

	// 2. Validate prefix "Salted__" (8 bytes)
	if len(ciphertext) < 16 || string(ciphertext[:8]) != "Salted__" {
		return "", fmt.Errorf("invalid CryptoJS format")
	}

	// 3. Extract the 8-byte salt
	salt := ciphertext[8:16]
	actualCiphertext := ciphertext[16:]
	// CBC decrypts whole blocks only, other data is not a CryptoJS text
	if len(actualCiphertext) == 0 || len(actualCiphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("invalid CryptoJS ciphertext: length=%d is not a multiple of %d", len(actualCiphertext), aes.BlockSize)
	}

	// 4. Derive Key and IV from passphrase and salt
	key, iv := c.deriveKeyAndIV([]byte(c.passphrase), salt)

	// 5. Initialize AES block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 6. Decrypt using CBC mode
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(actualCiphertext))
	mode.CryptBlocks(decrypted, actualCiphertext)

	// 7. Remove PKCS7 padding
	plaintext, err := c.pkcs7Unpad(decrypted)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (c *AesCrypt) Encrypt(plaintext string) (string, error) {
	// 1. Generate a random 8-byte salt
	salt := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// 2. Derive Key and IV using the same KDF as the decrypt function
	key, iv := c.deriveKeyAndIV([]byte(c.passphrase), salt)

	// 3. Apply PKCS7 padding to the plaintext
	// AES block size is 16 bytes
	paddedPlaintext := c.pkcs7Pad([]byte(plaintext), aes.BlockSize)

	// 4. Initialize AES block cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 5. Encrypt using CBC mode
	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// 6. Construct the CryptoJS format: "Salted__" + salt + ciphertext
	// "Salted__" is exactly```

	prefix := []byte("Salted__")
	finalBuffer := make([]byte, 0, len(prefix)+len(salt)+len(ciphertext))
	finalBuffer = append(finalBuffer, prefix...)
	finalBuffer = append(finalBuffer, salt...)
	finalBuffer = append(finalBuffer, ciphertext...)

	// 7. Return as Base64 string
	return base64.StdEncoding.EncodeToString(finalBuffer), nil
}

// DeriveKeyAndIV replicates OpenSSL's EVP_BytesToKey used by CryptoJS by default
func (c *AesCrypt) deriveKeyAndIV(passphrase []byte, salt []byte) (key []byte, iv []byte) {
	var concatenatedHashes []byte
	currentHash := []byte{}

	// CryptoJS standard AES-256-CBC needs 32 bytes for Key + 16 bytes for IV = 48 bytes total
	for len(concatenatedHashes) < 48 {
		hasher := md5.New()
		hasher.Write(currentHash)
		hasher.Write(passphrase)
		hasher.Write(salt)
		currentHash = hasher.Sum(nil)
		concatenatedHashes = append(concatenatedHashes, currentHash...)
	}

	return concatenatedHashes[:32], concatenatedHashes[32:48]
}

func (c *AesCrypt) pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7Unpad removes padding from the decrypted plaintext
func (c *AesCrypt) pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("data is empty")
	}
	padding := int(data[length-1])
	if padding < 1 || padding > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding size")
	}
	for i := length - padding; i < length; i++ {
		if int(data[i]) != padding {
			return nil, fmt.Errorf("invalid padding byte")
		}
	}
	return data[:length-padding], nil
}
