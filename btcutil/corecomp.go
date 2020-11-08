package btcutil

// Bitcoin Core Amount compression:
// * If the amount is 0, output 0, first, divide the amount (in base
//   units) by the largest power of 10 possible; call the exponent e
//   (e is max 9).
// * If e<9, the last digit of the resulting number cannot be 0;
//   store it as d, and drop it (divide by 10), call the result n,
//   and output 1+10*(9*n+d-1)+e.
// * If e==9, we only know the resulting number is not zero, so we
//   output 1+10*(n-1)+9, decodes: d is in [1-9] and e is in [0-9].
// XXX(jhj): NOt yet used, but please don't refactor out as dead!
func CompressAmount(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	var e uint64
	for (n%10) == 0 && e < 9 {
		n /= 10
		e++
	}
	if e < 9 {
		d := n % 10
		n /= 10
		return 1 + (n*9+d-1)*10 + e
	} else {
		return 1 + (n-1)*10 + 9
	}
}

// Bitcoin Core Amount Decompression
// XXX(jhj): Not yet used, but please don't refactor out as dead!
func DecompressAmount(x uint64) uint64 {
	if x == 0 {
		return 0
	}
	x--
	e := x % 10
	x /= 10
	var n uint64
	if e < 9 {
		d := (x % 9) + 1
		x /= 9
		n = x*10 + d
	} else {
		n = x + 1
	}
	for e != 0 {
		n *= 10
		e--
	}
	return n
}
