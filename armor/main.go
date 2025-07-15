package armor

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var (
	armorHeaderLine = regexp.MustCompile(`^-----BEGIN .+-----$`)
	armorHeader     = regexp.MustCompile(`^[a-zA-Z]+: .+$`)
	ErrArmorParse   = errors.New("error parsing ASCII armor format")
	ErrBadChecksum  = errors.New("calculated checksum does not match expected value")
)

const (
	ARMOR_PARSER_STATE_HEADERLINE int8 = iota
	ARMOR_PARSER_STATE_HEADER
	ARMOR_PARSER_STATE_BODY
	ARMOR_PARSER_STATE_FOOTER
	ARMOR_PARSER_STATE_DONE
)

// Convert the ASCII 'armored' PGP public key to a pure
// binary blob that can be written to disk or otherwise
// used for checking signatures.
//
// https://openpgp.dev/book/armor.html
func Parse(r io.Reader) (body []byte, err error) {
	var checksum uint32
	var state = ARMOR_PARSER_STATE_HEADERLINE
	var encodedKeyBuilder strings.Builder
	var expectedFooterLineText string
	scanner := bufio.NewScanner(r)
scanning:
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch state {
		case ARMOR_PARSER_STATE_HEADERLINE:
			if armorHeaderLine.MatchString(line) {
				expectedFooterLineText = strings.Replace(line, "BEGIN", "END", 1)
				state = ARMOR_PARSER_STATE_HEADER
				continue
			} else if line == "" {
				continue
			} else {
				err = fmt.Errorf("no header line found. Found '%v'", line)
				return
			}
		case ARMOR_PARSER_STATE_HEADER:
			if armorHeader.MatchString(line) {
				continue
			} else if line == "" {
				state = ARMOR_PARSER_STATE_BODY
				continue
			} else {
				err = fmt.Errorf("no end of header line. Found '%v'", line)
				return
			}
		case ARMOR_PARSER_STATE_BODY:
			if len(line) == 5 && line[0] == byte('=') {
				state = ARMOR_PARSER_STATE_FOOTER
				checksumBase64 := strings.TrimLeft(line, "=")
				var checksumBytes [4]byte
				var err2 error
				decoded, err2 := base64.StdEncoding.DecodeString(checksumBase64)
				if err2 != nil {
					err = err2
					return
				}
				copy(checksumBytes[1:4], decoded)
				checksum = binary.BigEndian.Uint32(checksumBytes[:])
				continue
			} else if line == expectedFooterLineText {
				state = ARMOR_PARSER_STATE_DONE
				continue
			}
			_, err = encodedKeyBuilder.WriteString(line)
			if err != nil {
				return
			}
		case ARMOR_PARSER_STATE_FOOTER:
			if line == expectedFooterLineText {
				state = ARMOR_PARSER_STATE_DONE
			} else {
				err = fmt.Errorf(`no footer line. Expected: "%v". Found "%v"`, expectedFooterLineText, line)
				return
			}
		case ARMOR_PARSER_STATE_DONE:
			// Discard rest of input
			break scanning
		}
	}
	if err = scanner.Err(); err != nil {
		return
	}
	if state != ARMOR_PARSER_STATE_DONE {
		err = fmt.Errorf("did not reach final parse state. Was in state %v", state)
		return
	}
	body, err = base64.StdEncoding.DecodeString(encodedKeyBuilder.String())
	if err != nil {
		return
	}
	// Checksums can be optional
	if checksum != 0 && checksum != crc24(body) {
		return []byte(""), ErrBadChecksum
	}
	return
}

const (
	CRC24_INIT  = 0xB704CE
	CRC24_GEN   = 0x1864CFB
	CRC24_WIDTH = 1 << 24
	CRC24_MASK  = CRC24_WIDTH - 1
)

// Calcualte the OpenPGP crc24 digest of a series of bytes
func crc24(data []byte) uint32 {
	var crc uint32 = CRC24_INIT
	for _, b := range data {
		crc ^= uint32(b) << 16
		for range 8 {
			crc <<= 1
			if crc&CRC24_WIDTH != 0 {
				crc ^= CRC24_GEN
			}
		}
	}
	return crc & CRC24_MASK
}
