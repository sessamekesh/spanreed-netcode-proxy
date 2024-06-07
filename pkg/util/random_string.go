package utils

import (
	"math/rand"
	"sync"
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go

type RandomStringGenerator struct {
	mut sync.Mutex
	gen rand.Source
}

func CreateRandomstringGenerator(seed int64) *RandomStringGenerator {
	src := rand.NewSource(seed)
	return &RandomStringGenerator{
		mut: sync.Mutex{},
		gen: src,
	}
}

var letters = []rune("123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ")

func (g *RandomStringGenerator) GetRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
