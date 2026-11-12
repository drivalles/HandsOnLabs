package lddata_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"ld-backend/internal/impl/core/data"
	"testing"

	"github.com/google/uuid"
)

func TestAes(t *testing.T) {
	key := uuid.NewString()
	c := lddata.NewAesCrypt(key)
	userId := "user1"
	userName := "User 1"
	companyId := "company1"
	locationId := "location1"
	m := make(map[string]string)
	m["userName"] = userName
	m["userId"] = userId
	m["companyId"] = companyId
	m["locationId"] = locationId
	text, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	encrypt, err := c.Encrypt(string(text))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Encrypted:", encrypt)
	decrypt, err := c.Decrypt(encrypt)
	if err != nil {
		t.Fatal(err)
	}
	if decrypt != string(text) {
		t.Error("Decryption failed")
	}
}

func TestAesDecryptInvalidCiphertext(t *testing.T) {
	c := lddata.NewAesCrypt(uuid.NewString())
	salted := func(ciphertext []byte) string {
		return base64.StdEncoding.EncodeToString(append([]byte("Salted__12345678"), ciphertext...))
	}
	inputs := map[string]string{
		"empty":                 "",
		"not base64":            "not base64 at all",
		"no salt prefix":        base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"salt without data":     salted(nil),
		"not full blocks":       salted(make([]byte, 17)),
		"less than a block":     salted(make([]byte, 5)),
		"one block of bad data": salted(make([]byte, 16)),
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Decrypt panicked: %v", r)
				}
			}()
			if _, err := c.Decrypt(input); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
