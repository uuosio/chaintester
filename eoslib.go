package chaintester

var (
	charmap = []byte(".12345abcdefghijklmnopqrstuvwxyz")
)

func N2S(value uint64) string {
	// 13 dots
	str := []byte{'.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.'}

	tmp := value
	for i := 0; i <= 12; i++ {
		var c byte
		if i == 0 {
			c = charmap[tmp&0x0f]
		} else {
			c = charmap[tmp&0x1f]
		}
		str[12-i] = c
		if i == 0 {
			tmp >>= 4
		} else {
			tmp >>= 5
		}
	}

	i := len(str) - 1
	for ; i >= 0; i-- {
		if str[i] != '.' {
			break
		}
	}
	return string(str[:i+1])
}
