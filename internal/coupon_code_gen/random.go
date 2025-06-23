package coupon_code_gen

import (
	"context"
	"coupon_issuance/internal/datastore"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

func allKoreanRunes() []rune {
	var runes []rune
	for r := rune(0xAC00); r <= rune(0xD7A3); r++ {
		runes = append(runes, r)
	}
	return runes
}

var numberRunes = []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
var allRunes = append(allKoreanRunes(), numberRunes...)

const codeLength = 10

type RandomCouponCodeGenerator struct {
	rnd       *rand.Rand
	DataStore *datastore.RedisDataStore
	mu        sync.Mutex
}

func NewRandomCouponCodeGenerator(ds *datastore.RedisDataStore) *RandomCouponCodeGenerator {
	return &RandomCouponCodeGenerator{
		rnd:       rand.New(rand.NewSource(time.Now().UnixNano())),
		DataStore: ds,
	}
}

func (g *RandomCouponCodeGenerator) Generate(ctx context.Context) (string, error) {
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		code := make([]rune, codeLength)
		for j := 0; j < codeLength; j++ {
			g.mu.Lock()
			code[j] = allRunes[g.rnd.Intn(len(allRunes))]
			g.mu.Unlock()
		}

		codeStr := string(code)
		exists, err := g.DataStore.CheckCouponCodeExists(ctx, codeStr)
		if err != nil {
			return "", err
		}
		if !exists {
			return codeStr, nil
		}
	}
	log.Printf("CRITICAL: Failed to generate unique coupon code after %d attempts", maxAttempts)
	return "", errors.New("failed to generate unique coupon code")
} 