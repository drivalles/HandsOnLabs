package lddata

import (
	"fmt"
	"strings"
	"unicode"
)

func KeepDigitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func ValidatePhoneNumber(input string) (string, error) {
	phoneNumber := KeepDigitsOnly(input)
	if phoneNumber == "" {
		return "", fmt.Errorf("empty phone number %s", input)
	}
	if len(phoneNumber) == 10 {
		phoneNumber = "1" + phoneNumber
	}
	if phoneNumber[0] != '1' {
		return "", fmt.Errorf("phone does not starts with 1 %s", phoneNumber)
	}
	if len(phoneNumber) != 11 {
		return "", fmt.Errorf("phone does not have 11 digits %s", phoneNumber)
	}
	return phoneNumber, nil
}
