package main

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/text/encoding/charmap"
)

// ByteReader читатель бинарных данных
type ByteReader struct {
	data   []byte
	offset int
}

// NewByteReader создает новый ByteReader
func NewByteReader(data []byte) *ByteReader {
	return &ByteReader{
		data:   data,
		offset: 0,
	}
}

// ReadByte читает один байт
func (br *ByteReader) ReadByte() (byte, error) {
	if br.offset+1 > len(br.data) {
		return 0, fmt.Errorf("EOF")
	}
	result := br.data[br.offset]
	br.offset++
	return result, nil
}

// ReadWord читает 16-битное значение little endian
func (br *ByteReader) ReadWord() (uint16, error) {
	if br.offset+2 > len(br.data) {
		return 0, fmt.Errorf("EOF")
	}
	result := binary.LittleEndian.Uint16(br.data[br.offset:])
	br.offset += 2
	return result, nil
}

// ReadDWord читает 32-битное значение little endian
func (br *ByteReader) ReadDWord() (uint32, error) {
	if br.offset+4 > len(br.data) {
		return 0, fmt.Errorf("EOF")
	}
	result := binary.LittleEndian.Uint32(br.data[br.offset:])
	br.offset += 4
	return result, nil
}

// ReadBytes читает указанное количество байт
func (br *ByteReader) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		n = len(br.data) - br.offset // Читаем до конца
	}
	if br.offset+n > len(br.data) {
		return nil, fmt.Errorf("EOF")
	}
	result := make([]byte, n)
	copy(result, br.data[br.offset:br.offset+n])
	br.offset += n
	return result, nil
}

// ReadName читает строку с префиксом длины (1 байт) в кодировке CP1251
func (br *ByteReader) ReadName() (string, error) {
	nameLen, err := br.ReadByte()
	if err != nil {
		return "", err
	}
	
	nameBytes, err := br.ReadBytes(int(nameLen))
	if err != nil {
		return "", err
	}
	
	// Декодируем из CP1251
	decoder := charmap.Windows1251.NewDecoder()
	result, err := decoder.Bytes(nameBytes)
	if err != nil {
		return string(nameBytes), nil // Fallback к raw bytes
	}
	
	return string(result), nil
}

// ReadLongString читает строку с префиксом длины (4 байта) в кодировке CP1251
func (br *ByteReader) ReadLongString() (string, error) {
	nameLen, err := br.ReadDWord()
	if err != nil {
		return "", err
	}
	
	nameBytes, err := br.ReadBytes(int(nameLen))
	if err != nil {
		return "", err
	}
	
	// Декодируем из CP1251
	decoder := charmap.Windows1251.NewDecoder()
	result, err := decoder.Bytes(nameBytes)
	if err != nil {
		return string(nameBytes), nil // Fallback к raw bytes
	}
	
	return string(result), nil
}

// EOF проверяет достигнут ли конец данных
func (br *ByteReader) EOF() bool {
	return br.offset >= len(br.data)
}

// Remaining возвращает оставшиеся байты
func (br *ByteReader) Remaining() []byte {
	if br.offset >= len(br.data) {
		return []byte{}
	}
	return br.data[br.offset:]
}