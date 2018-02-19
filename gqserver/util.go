package gqserver

// Because Uint methods from binary package are stupid
func BtoInt(b []byte) int {
	var mult uint = 1
	var sum uint = 0
	length := uint(len(b))
	var i uint
	for i = 0; i < length; i++ {
		sum += uint(b[i]) * (mult << ((length - i - 1) * 8))
	}
	return int(sum)
}
