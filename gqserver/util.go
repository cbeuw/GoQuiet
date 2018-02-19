package gqserver

// BtoInt converts a byte slice into int in Big Endian order
// Uint methods from binary package can be used, but they are messy
func BtoInt(b []byte) int {
	var mult uint = 1
	var sum uint
	length := uint(len(b))
	var i uint
	for i = 0; i < length; i++ {
		sum += uint(b[i]) * (mult << ((length - i - 1) * 8))
	}
	return int(sum)
}
