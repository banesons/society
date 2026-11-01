package formula

import (
	"errors"
	"math/big"
	"strings"
)

var (
	ErrDivisionByZero = errors.New("division by zero")
)

// Decimal represents an exact rational number for deterministic financial arithmetic.
type Decimal struct {
	r *big.Rat
}

// Zero returns a Decimal initialized to 0.
func Zero() Decimal {
	return Decimal{r: big.NewRat(0, 1)}
}

// One returns a Decimal initialized to 1.
func One() Decimal {
	return Decimal{r: big.NewRat(1, 1)}
}

// NewDecimalFromInt creates a Decimal from an int64.
func NewDecimalFromInt(val int64) Decimal {
	return Decimal{r: big.NewRat(val, 1)}
}

// NewDecimalFromFloat creates a Decimal from a float64.
func NewDecimalFromFloat(val float64) Decimal {
	r := new(big.Rat)
	r.SetFloat64(val)
	return Decimal{r: r}
}

// NewDecimalFromString parses a decimal string (e.g. "865", "8.52", "-12.34").
func NewDecimalFromString(val string) Decimal {
	trimmed := strings.TrimSpace(val)
	r := new(big.Rat)
	if _, ok := r.SetString(trimmed); !ok {
		// Fallback for empty or invalid
		return Zero()
	}
	return Decimal{r: r}
}

// Add returns d + other.
func (d Decimal) Add(other Decimal) Decimal {
	res := new(big.Rat).Add(d.rat(), other.rat())
	return Decimal{r: res}
}

// Sub returns d - other.
func (d Decimal) Sub(other Decimal) Decimal {
	res := new(big.Rat).Sub(d.rat(), other.rat())
	return Decimal{r: res}
}

// Mul returns d * other.
func (d Decimal) Mul(other Decimal) Decimal {
	res := new(big.Rat).Mul(d.rat(), other.rat())
	return Decimal{r: res}
}

// Div returns d / other, returning ErrDivisionByZero if other == 0.
func (d Decimal) Div(other Decimal) (Decimal, error) {
	if other.IsZero() {
		return Zero(), ErrDivisionByZero
	}
	res := new(big.Rat).Quo(d.rat(), other.rat())
	return Decimal{r: res}, nil
}

// Neg returns -d.
func (d Decimal) Neg() Decimal {
	res := new(big.Rat).Neg(d.rat())
	return Decimal{r: res}
}

// Abs returns |d|.
func (d Decimal) Abs() Decimal {
	res := new(big.Rat).Abs(d.rat())
	return Decimal{r: res}
}

// Cmp compares d and other: -1 if d < other, 0 if d == other, +1 if d > other.
func (d Decimal) Cmp(other Decimal) int {
	return d.rat().Cmp(other.rat())
}

// Equal returns true if d == other.
func (d Decimal) Equal(other Decimal) bool {
	return d.Cmp(other) == 0
}

// GreaterThan returns true if d > other.
func (d Decimal) GreaterThan(other Decimal) bool {
	return d.Cmp(other) > 0
}

// GreaterThanOrEqual returns true if d >= other.
func (d Decimal) GreaterThanOrEqual(other Decimal) bool {
	return d.Cmp(other) >= 0
}

// LessThan returns true if d < other.
func (d Decimal) LessThan(other Decimal) bool {
	return d.Cmp(other) < 0
}

// LessThanOrEqual returns true if d <= other.
func (d Decimal) LessThanOrEqual(other Decimal) bool {
	return d.Cmp(other) <= 0
}

// IsZero returns true if d == 0.
func (d Decimal) IsZero() bool {
	return d.rat().Sign() == 0
}

// IsPositive returns true if d > 0.
func (d Decimal) IsPositive() bool {
	return d.rat().Sign() > 0
}

// IsNegative returns true if d < 0.
func (d Decimal) IsNegative() bool {
	return d.rat().Sign() < 0
}

// Floor returns the largest integer less than or equal to d.
func (d Decimal) Floor() Decimal {
	num := d.rat().Num()
	denom := d.rat().Denom()
	q := new(big.Int).Div(num, denom)
	return Decimal{r: new(big.Rat).SetInt(q)}
}

// Ceil returns the smallest integer greater than or equal to d.
func (d Decimal) Ceil() Decimal {
	num := d.rat().Num()
	denom := d.rat().Denom()
	q, r := new(big.Int).QuoRem(num, denom, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return Decimal{r: new(big.Rat).SetInt(q)}
}

// Round returns d rounded to places decimal places using Half-Up rounding.
func (d Decimal) Round(places int) Decimal {
	if places < 0 {
		places = 0
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	scaleRat := new(big.Rat).SetInt(scale)

	// scaled = d * 10^places
	scaled := new(big.Rat).Mul(d.rat(), scaleRat)

	half := big.NewRat(1, 2)
	var roundedInt *big.Int

	if scaled.Sign() >= 0 {
		// scaled + 0.5 -> floor
		shifted := new(big.Rat).Add(scaled, half)
		roundedInt = new(big.Int).Quo(shifted.Num(), shifted.Denom())
	} else {
		// scaled - 0.5 -> ceil
		shifted := new(big.Rat).Sub(scaled, half)
		q, r := new(big.Int).QuoRem(shifted.Num(), shifted.Denom(), new(big.Int))
		if r.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		}
		roundedInt = q
	}

	resultRat := new(big.Rat).Quo(new(big.Rat).SetInt(roundedInt), scaleRat)
	return Decimal{r: resultRat}
}

// String returns a clean decimal string without trailing zeroes where unnecessary,
// or as an exact decimal representation.
func (d Decimal) String() string {
	r := d.rat()
	if r.IsInt() {
		return r.Num().String()
	}

	// Format to 10 decimal places, trim trailing zeroes
	str := r.FloatString(10)
	if strings.Contains(str, ".") {
		str = strings.TrimRight(str, "0")
		str = strings.TrimRight(str, ".")
	}
	return str
}

// FormatCurrency returns the value rounded to 2 decimal places with exact 2 decimals.
func (d Decimal) FormatCurrency() string {
	rounded := d.Round(2)
	return rounded.rat().FloatString(2)
}

// Float64 converts decimal to float64 (for external consumers).
func (d Decimal) Float64() float64 {
	f, _ := d.rat().Float64()
	return f
}

func (d Decimal) rat() *big.Rat {
	if d.r == nil {
		return big.NewRat(0, 1)
	}
	return d.r
}
