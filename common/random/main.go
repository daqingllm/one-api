package random

import (
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GetUUID() string {
	code := uuid.New().String()
	code = strings.Replace(code, "-", "", -1)
	return code
}

const keyChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const keyNumbers = "0123456789"
const usernameCharset = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKMNOPQRSTUVWXYZ23456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateKey() string {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, 48)
	for i := 0; i < 16; i++ {
		key[i] = keyChars[rand.Intn(len(keyChars))]
	}
	uuid_ := GetUUID()
	for i := 0; i < 32; i++ {
		c := uuid_[i]
		if i%2 == 0 && c >= 'a' && c <= 'z' {
			c = c - 'a' + 'A'
		}
		key[i+16] = c
	}
	return string(key)
}

func GetRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, length)
	for i := 0; i < length; i++ {
		key[i] = keyChars[rand.Intn(len(keyChars))]
	}
	return string(key)
}

func GetRandomNumberString(length int) string {
	rand.Seed(time.Now().UnixNano())
	key := make([]byte, length)
	for i := 0; i < length; i++ {
		key[i] = keyNumbers[rand.Intn(len(keyNumbers))]
	}
	return string(key)
}

// RandRange returns a random number between min and max (max is not included)
func RandRange(min, max int) int {
	return min + rand.Intn(max-min)
}

func GenerateRandomUsername() string {
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano())) // 独立随机源
	code := make([]byte, 6)
	for i := range code {
		code[i] = usernameCharset[randSrc.Intn(len(usernameCharset))]
	}
	return "fun_" + string(code)
}
