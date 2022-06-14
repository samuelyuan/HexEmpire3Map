package fileio

const (
	HLOG  = 14
	HSIZE = 1 << HLOG
	// The maximum number of literals in a chunk (32)
	MAX_LIT = 1 << 5
	MAX_OFF = 1 << 13
	// The maximum back-reference length (264).
	MAX_REF = (1 << 8) + (1 << 3)
)

var (
	HashTable = make([]uint64, HSIZE)
)

// This function will compress the file correctly such that the decompress will generate a map,
// but it does not generate the same file that the game does
func Compress(inputBytes []byte) []byte {
	length := len(inputBytes) * 2
	output := make([]byte, length)
	count := LzfCompress(inputBytes, output)
	for {
		if count != 0 {
			break
		}

		length *= 2
		output = make([]byte, length)
		count = LzfCompress(inputBytes, output)
	}
	dst := make([]byte, count)
	for i := 0; i < count; i++ {
		dst[i] = output[i]
	}
	return dst
}

func Decompress(inputBytes []byte) []byte {
	length := len(inputBytes) * 2
	output := make([]byte, length)
	count := LzfDecompress(inputBytes, output)

	for {
		// Decompress succeeded
		if count != 0 {
			break
		}

		// If initial array is too small, double size and try again
		length *= 2
		output = make([]byte, length)
		count = LzfDecompress(inputBytes, output)
	}
	dst := make([]byte, count)
	for i := 0; i < count; i++ {
		dst[i] = output[i]
	}
	return dst
}

func getHashSlot(hashValue uint64) uint64 {
	return ((hashValue^hashValue<<5)>>(24-HLOG) - hashValue*5) & (HSIZE - 1)
}

func LzfCompress(input []byte, output []byte) int {
	var hval, hashSlot, reference, offset uint64

	inputLength := len(input)
	outputLength := len(output)
	for i := 0; i < HSIZE; i++ {
		HashTable[i] = 0
	}
	inputIndex := 0
	outputIndex := 0
	lit := 0

	hval = (uint64(input[inputIndex]) << 8) | uint64(input[inputIndex+1])

	for {
		if inputIndex < inputLength-2 {
			hval = (hval << 8) | uint64(input[inputIndex+2])
			hashSlot = getHashSlot(hval)
			reference = HashTable[hashSlot]
			HashTable[hashSlot] = uint64(inputIndex)
			offset = uint64(inputIndex) - reference - 1

			if offset < MAX_OFF &&
				inputIndex+4 < inputLength &&
				reference > 0 &&
				input[reference] == input[inputIndex] &&
				input[reference+1] == input[inputIndex+1] &&
				input[reference+2] == input[inputIndex+2] {
				length := 2
				maxLength := inputLength - inputIndex - length
				if maxLength > MAX_REF {
					maxLength = MAX_REF
				}
				if int64(outputIndex)+int64(lit)+1+3 >= int64(outputLength) {
					return 0
				}
				for {
					length++
					if length >= maxLength || input[int(reference)+length] != input[inputIndex+length] {
						break
					}
				}
				if lit != 0 {
					output[outputIndex] = byte(lit - 1)
					outputIndex++
					for lit = -lit; lit != 0; lit++ {
						output[outputIndex] = input[int64(inputIndex)+int64(lit)]
						outputIndex++
					}
				}
				length -= 2
				inputIndex++
				if length < 7 {
					output[outputIndex] = byte(uint64(offset>>8) + uint64(length<<5))
					outputIndex++
				} else {
					output[outputIndex] = byte(uint64(offset>>8) + (7 << 5))
					output[outputIndex+1] = byte(length - 7)
					outputIndex += 2
				}
				output[outputIndex] = byte(offset)
				outputIndex++

				inputIndex += length - 1

				hval = (uint64(input[inputIndex]) << 8) | uint64(input[inputIndex+1])
				hval = (hval << 8) | uint64(input[inputIndex+2])
				hashSlot = getHashSlot(hval)
				HashTable[hashSlot] = uint64(inputIndex)
				inputIndex++

				hval = (hval << 8) | uint64(input[inputIndex+2])
				hashSlot = getHashSlot(hval)
				HashTable[hashSlot] = uint64(inputIndex)
				inputIndex++
				continue
			}
		} else if inputIndex == inputLength {
			break
		}
		// Copy one more byte
		lit++
		inputIndex++

		if int64(lit) == int64(MAX_LIT) {
			if int64(outputIndex+1+MAX_LIT) >= int64(outputLength) {
				return 0
			}
			output[outputIndex] = byte(MAX_LIT - 1)
			outputIndex++

			for lit = -lit; lit != 0; lit++ {
				output[outputIndex] = input[inputIndex+lit]
				outputIndex++
			}
		}
	}

	if lit != 0 {
		if int64(outputIndex)+int64(lit)+1 >= int64(outputLength) {
			return 0
		}
		output[outputIndex] = byte(lit - 1)
		outputIndex++
		for lit = -lit; lit != 0; lit++ {
			output[outputIndex] = input[inputIndex+lit]
			outputIndex++
		}
	}
	return int(outputIndex)
}

func LzfDecompress(input []byte, output []byte) int {
	inputLength := len(input)
	outputLength := len(output)
	inputIndex := uint(0)
	outputIndex := uint(0)
	for {
		inputByte := uint(input[inputIndex])
		inputIndex++
		if inputByte < 32 {
			dataLength := uint(inputByte) + 1
			// Make sure output array bounds have enough space
			if int64(outputIndex+dataLength) > int64(outputLength) {
				// Not enough space
				return 0
			}

			// Copy the data from input array into output array
			for i := 0; i < int(dataLength); i++ {
				output[int(outputIndex)+i] = input[int(inputIndex)+i]
			}
			outputIndex += dataLength
			inputIndex += dataLength
		} else {
			dataLength := uint(inputByte >> 5)
			newIndex := int(outputIndex) - ((int(inputByte) & 31) << 8) - 1
			if dataLength == 7 {
				dataLength += uint(input[inputIndex])
				inputIndex++
			}
			reference := newIndex - int(input[inputIndex])
			inputIndex++
			if int64(uint(int(outputIndex)+int(dataLength)+2)) > int64(outputLength) {
				// Array out of bounds
				return 0
			}

			if reference < 0 {
				return 0
			}
			// Copy data
			for i := 0; i < int(dataLength)+2; i++ {
				output[int(outputIndex)+i] = output[reference+i]
			}
			outputIndex += dataLength + 2
		}
		// Done processing input array
		if int64(inputIndex) >= int64(inputLength) {
			break
		}
	}
	return int(outputIndex)
}
