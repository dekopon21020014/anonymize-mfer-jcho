package mfer

import (
	"encoding/binary"
	"errors"
	"fmt"
)

func Anonymize(bytes []byte) ([]byte, error) {
	var (
		tagCode byte
		length  uint32
	)

	for i := 0; i < len(bytes); {
		tagCode = bytes[i]
		i++
		if tagCode == ZERO {
			continue
		} else if tagCode == END {
			break
		}

		length = uint32(bytes[i])
		i++

		if length > 0x7f { /* MSBが1ならば */
			numBytes := length - 0x80
			if numBytes > 4 {
				fmt.Println("byets = ", bytes[i-2:i+15])
				fmt.Printf("length = %x, numBytes = %d, bytes = %d\n", length, numBytes, bytes[i-1])
				return bytes, errors.New("error nbytes")
			}
			length = binary.BigEndian.Uint32(append(make([]byte, 4-numBytes), bytes[i:i+int(numBytes)]...))
			i += int(numBytes)
		}

		switch tagCode {
		case CHANNEL_ATTRIBUTE:
			length = uint32(bytes[i])
			i++

		// about patient
		case P_NAME:
			bytes = append(bytes[:i-2], bytes[i+int(length):]...)
			i -= 2
			continue

		case P_ID:
			bytes = append(bytes[:i-2], bytes[i+int(length):]...)
			i -= 2
			continue

		case P_AGE:
			bytes = append(bytes[:i-2], bytes[i+int(length):]...)
			i -= 2
			continue

		case P_SEX:
			// do nothing
		}
		i += int(length)
	}
	return bytes, nil
}
