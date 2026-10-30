package utils

import (
	"math/rand"
	"sync"
)

var unique = sync.Map{}

func GenerateRandomString(length int) string {
	randomString := doGenerateRandomString(length)
	count := 1

	for {
		if _, exists := unique.Load(randomString); !exists {
			unique.Store(randomString, true)
			return randomString
		}

		randomString = doGenerateRandomString(length)
		count++
		if count >= 1000 {
			return randomString
		}
	}
}

func doGenerateRandomString(n int) string {
	if n <= 0 {
		return ""
	}

	// A slice of characters to pick from
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// Create a byte slice with space for n characters
	b := make([]rune, n)

	for i := range b {
		// rand.IntN safely picks a random number up to the length of letters
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func RandomElement[T any](array []T) (T, bool) {
	var zero T
	if len(array) == 0 {
		return zero, false
	}
	return array[rand.Intn(len(array))], true
}

func RandomElements[T any](array []T, max int) []T {
	if len(array) == 0 || max <= 0 {
		return []T{}
	}

	// randomCount in [1, max]
	randomCount := rand.Intn(max) + 1
	result := make([]T, 0, randomCount)

	current := make([]T, len(array))
	copy(current, array)
	for i := 0; i < randomCount; i++ {
		if len(current) == 0 {
			break
		}
		randomIndex := rand.Intn(len(current))
		result = append(result, current[randomIndex])
		current = append(current[:randomIndex], current[randomIndex+1:]...)
	}

	return result
}

func RandomInt(min, max int) int {
	if max < min {
		min, max = max, min
	}
	return rand.Intn(max-min+1) + min
}

func RandomBool(probability ...float64) bool {
	p := 0.5
	if len(probability) > 0 {
		p = probability[0]
	}
	return rand.Float64() < p
}
