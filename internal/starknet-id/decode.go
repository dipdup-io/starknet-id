package starknetid

import (
	"strings"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	"github.com/shopspring/decimal"
)

const (
	basicAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789-"
	bigAlphabet   = "这来"
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
		bigLen   = decimal.NewFromInt(2)
		basicL   = basicLen.Add(one)
		bigL     = bigLen.Add(one)
	)

	for num.IsPositive() {
		var char byte

		code := num.Mod(basicL)
		num = num.Div(basicL).Floor()

		if code.Equal(basicLen) {
			nextFelt := num.Div(bigL).Floor()
			if nextFelt.IsZero() {
				code2 := num.Div(bigL).Floor()
				num = nextFelt
				if code2.IsZero() {
					char = basicAlphabet[0]
				} else {
					char = bigAlphabet[code2.Sub(one).IntPart()]
				}
			} else {
				index := num.Mod(bigLen).IntPart()
				char = bigAlphabet[index]
				num = num.Div(bigLen).Floor()
			}
		} else {
			char = basicAlphabet[code.IntPart()]
		}

		if err := decoded.WriteByte(char); err != nil {
			return "", err
		}
	}

	decodedString, k := extractStars(decoded.String())
	if k > 0 {
		var (
			first    = bigAlphabet[0]
			last     = bigAlphabet[len(bigAlphabet)-1]
			basicSym = basicAlphabet[1]
			char     byte
		)
		if k%2 == 0 {
			char = (last * byte(k/2-1)) + first + basicSym
		} else {
			char = last * byte(k/2+1)
		}

		decodedString += string(char)
	}

	return decodedString, nil
}
