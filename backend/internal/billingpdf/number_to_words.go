package billingpdf

import (
	"fmt"
	"math"
	"strings"
)

var (
	ones = []string{
		"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
		"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen",
		"Seventeen", "Eighteen", "Nineteen",
	}

	tens = []string{
		"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
	}
)

// NumberToWords converts a monetary amount to Indian English currency representation.
// E.g., 7369.80 -> "Rupees Seven Thousand Three Hundred Sixty-Nine and Eighty Paise Only"
func NumberToWords(amount float64) string {
	if math.Abs(amount) < 0.005 {
		return "Rupees Zero Only"
	}

	isNegative := amount < 0
	if isNegative {
		amount = math.Abs(amount)
	}

	// Split integer rupees and fractional paise (rounded to 2 decimals)
	totalPaise := int64(math.Round(amount * 100))
	rupees := totalPaise / 100
	paise := totalPaise % 100

	var parts []string

	if isNegative {
		parts = append(parts, "Minus")
	}

	parts = append(parts, "Rupees")

	if rupees == 0 {
		if paise > 0 {
			parts = append(parts, "Zero and", convertTwoDigits(int(paise)), "Paise Only")
			return cleanWordString(strings.Join(parts, " "))
		}
		return "Rupees Zero Only"
	}

	rupeesWords := convertIndianRupees(rupees)
	parts = append(parts, rupeesWords)

	if paise > 0 {
		parts = append(parts, "and", convertTwoDigits(int(paise)), "Paise Only")
	} else {
		parts = append(parts, "Only")
	}

	return cleanWordString(strings.Join(parts, " "))
}

func convertIndianRupees(n int64) string {
	if n == 0 {
		return "Zero"
	}

	var parts []string

	// Crores (1,00,00,000)
	if n >= 10000000 {
		crores := n / 10000000
		parts = append(parts, convertIndianRupees(crores)+" Crore")
		n %= 10000000
	}

	// Lakhs (1,00,000)
	if n >= 100000 {
		lakhs := n / 100000
		parts = append(parts, convertTwoDigits(int(lakhs))+" Lakh")
		n %= 100000
	}

	// Thousands (1,000)
	if n >= 1000 {
		thousands := n / 1000
		parts = append(parts, convertTwoDigits(int(thousands))+" Thousand")
		n %= 1000
	}

	// Hundreds (100)
	if n >= 100 {
		hundreds := n / 100
		parts = append(parts, ones[hundreds]+" Hundred")
		n %= 100
	}

	// Remainder (0 - 99)
	if n > 0 {
		parts = append(parts, convertTwoDigits(int(n)))
	}

	return strings.Join(parts, " ")
}

func convertTwoDigits(n int) string {
	if n < 20 {
		return ones[n]
	}
	t := tens[n/10]
	o := ones[n%10]
	if o != "" {
		return t + "-" + o
	}
	return t
}

func cleanWordString(s string) string {
	words := strings.Fields(s)
	return strings.Join(words, " ")
}

// FormatINR formats a float64 into standard Indian Rupee notation (e.g. 12,34,567.89).
func FormatINR(amount float64) string {
	isNeg := amount < 0
	if isNeg {
		amount = math.Abs(amount)
	}

	totalPaise := int64(math.Round(amount * 100))
	intPart := totalPaise / 100
	fracPart := totalPaise % 100

	intStr := fmt.Sprintf("%d", intPart)
	var formattedInt string

	if len(intStr) <= 3 {
		formattedInt = intStr
	} else {
		// Last 3 digits
		last3 := intStr[len(intStr)-3:]
		rest := intStr[:len(intStr)-3]

		var chunks []string
		for len(rest) > 2 {
			chunk := rest[len(rest)-2:]
			chunks = append([]string{chunk}, chunks...)
			rest = rest[:len(rest)-2]
		}
		if len(rest) > 0 {
			chunks = append([]string{rest}, chunks...)
		}
		formattedInt = strings.Join(chunks, ",") + "," + last3
	}

	res := fmt.Sprintf("%s.%02d", formattedInt, fracPart)
	if isNeg {
		res = "-" + res
	}
	return res
}
