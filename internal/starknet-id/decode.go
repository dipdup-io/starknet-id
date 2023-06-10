package starknetid

import (
	"strings"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	"github.com/shopspring/decimal"
)

const (
	basicAlphabet     = "abcdefghijklmnopqrstuvwxyz0123456789-"
	bigAlphabetString = "这来"
)

var (
	bigAlphabet = []rune(bigAlphabetString)
)

func extractStars(s string) (string, int) {
	result := strings.TrimRight(s, string(bigAlphabet[len(bigAlphabet)-1]))
	return result, len(s) - len(result)
}

// Decode -
func Decode(f data.Felt) (string, error) {
	num := f.Decimal()

	var (
		decoded = new(strings.Builder)

		one      = decimal.NewFromInt(1)
		basicLen = decimal.NewFromInt(int64(len(basicAlphabet)))
		bigLen   = decimal.NewFromInt(int64(len(bigAlphabet)))
		basicL   = basicLen.Add(one)
		bigL     = bigLen.Add(one)
	)

	for num.IsPositive() {
		var char rune

		code := num.Mod(basicL)
		num = num.Div(basicL).Floor()

		if code.Equal(basicLen) {
			nextFelt := num.Div(bigL).Floor()
			if nextFelt.IsZero() {
				code2 := num.Div(bigL).Floor()
				num = nextFelt
				if code2.IsZero() {
					char = rune(basicAlphabet[0])
				} else {
					char = bigAlphabet[code2.Sub(one).IntPart()]
				}
			} else {
				index := num.Mod(bigLen).BigInt().Int64()
				char = bigAlphabet[index]
				num = num.Div(bigLen).Floor()
			}
		} else {
			char = rune(basicAlphabet[code.IntPart()])
		}

		if _, err := decoded.WriteRune(char); err != nil {
			return "", err
		}
	}

	decodedString, k := extractStars(decoded.String())
	if k > 0 {
		var (
			first    = bigAlphabet[0]
			last     = bigAlphabet[len(bigAlphabet)-1]
			basicSym = rune(basicAlphabet[1])
			char     rune
		)
		if k%2 == 0 {
			char = (last * rune(k/2-1)) + first + basicSym
		} else {
			char = last * rune(k/2+1)
		}

		decodedString += string(char)
	}

	return decodedString, nil
}
