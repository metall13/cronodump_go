package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FieldType представляет тип поля в таблице Kronos
type FieldType int

const (
	FieldTypeSystemNumber FieldType = 0 // Системный номер (PRIMARY KEY)
	FieldTypeInteger      FieldType = 1 // Целое число
	FieldTypeString       FieldType = 2 // Строка (VARCHAR)
	FieldTypeText         FieldType = 3 // Текст (TEXT)
	FieldTypeDate         FieldType = 4 // Дата
	FieldTypeTime         FieldType = 5 // Время/Timestamp
	FieldTypeFile         FieldType = 6 // Ссылка на файл
	FieldTypeForeignKey7  FieldType = 7 // Внешний ключ
	FieldTypeForeignKey8  FieldType = 8 // Внешний ключ
	FieldTypeForeignKey9  FieldType = 9 // Внешний ключ
)

// FieldDefinition описывает структуру поля в таблице
type FieldDefinition struct {
	Type     FieldType
	Index1   uint32
	Name     string
	Flags    uint32
	MinVal   byte
	Index2   uint32
	MaxVal   uint32
	Unknown4 uint32
}

// TableDefinition описывает структуру таблицы
type TableDefinition struct {
	TableID   uint32
	TableName string
	Abbrev    string
	Fields    []FieldDefinition
	Version   byte
}

// Record представляет запись в таблице
type Record struct {
	SystemNumber uint32
	Fields       []string
}

// KODDecoder декодер для расшифровки данных Kronos
type KODDecoder struct {
	kodTable [256]byte
	invTable [256]byte
}

// NewKODDecoder создает новый декодер с стандартной таблицей KOD
func NewKODDecoder() *KODDecoder {
	// Стандартная таблица KOD из cronodump
	initialKOD := [256]byte{
		0x08, 0x63, 0x81, 0x38, 0xA3, 0x6B, 0x82, 0xA6, 0x18, 0x0D, 0xAC, 0xD5, 0xFE, 0xBE, 0x15, 0xF6,
		0xA5, 0x36, 0x76, 0xE2, 0x2D, 0x41, 0xB5, 0x12, 0x4B, 0xD8, 0x3C, 0x56, 0x34, 0x46, 0x4F, 0xA4,
		0xD0, 0x01, 0x8B, 0x60, 0x0F, 0x70, 0x57, 0x3E, 0x06, 0x67, 0x02, 0x7A, 0xF8, 0x8C, 0x80, 0xE8,
		0xC3, 0xFD, 0x0A, 0x3A, 0xA7, 0x73, 0xB0, 0x4D, 0x99, 0xA2, 0xF1, 0xFB, 0x5A, 0xC7, 0xC2, 0x17,
		0x96, 0x71, 0xBA, 0x2A, 0xA9, 0x9A, 0xF3, 0x87, 0xEA, 0x8E, 0x09, 0x9E, 0xB9, 0x47, 0xD4, 0x97,
		0xE4, 0xB3, 0xBC, 0x58, 0x53, 0x5F, 0x2E, 0x21, 0xD1, 0x1A, 0xEE, 0x2C, 0x64, 0x95, 0xF2, 0xB8,
		0xC6, 0x33, 0x8D, 0x2B, 0x1F, 0xF7, 0x25, 0xAD, 0xFF, 0x7F, 0x39, 0xA8, 0xBF, 0x6A, 0x91, 0x79,
		0xED, 0x20, 0x7B, 0xA1, 0xBB, 0x45, 0x69, 0xCD, 0xDC, 0xE7, 0x31, 0xAA, 0xF0, 0x65, 0xD7, 0xA0,
		0x32, 0x93, 0xB1, 0x24, 0xD6, 0x5B, 0x9F, 0x27, 0x42, 0x85, 0x07, 0x44, 0x3F, 0xB4, 0x11, 0x68,
		0x5E, 0x49, 0x29, 0x13, 0x94, 0xE6, 0x1B, 0xE1, 0x7D, 0xC8, 0x2F, 0xFA, 0x78, 0x1D, 0xE3, 0xDE,
		0x50, 0x4E, 0x89, 0xB6, 0x30, 0x48, 0x0C, 0x10, 0x05, 0x43, 0xCE, 0xD3, 0x61, 0x51, 0x83, 0xDA,
		0x77, 0x6F, 0x92, 0x9D, 0x74, 0x7C, 0x04, 0x88, 0x86, 0x55, 0xCA, 0xF4, 0xC1, 0x62, 0x0E, 0x28,
		0xB7, 0x0B, 0xC0, 0xF5, 0xCF, 0x35, 0xC5, 0x4C, 0x16, 0xE0, 0x98, 0x00, 0x9B, 0xD9, 0xAE, 0x03,
		0xAF, 0xEC, 0xC9, 0xDB, 0x6D, 0x3B, 0x26, 0x75, 0x3D, 0xBD, 0xB2, 0x4A, 0x5D, 0x6C, 0x72, 0x40,
		0x7E, 0xAB, 0x59, 0x52, 0x54, 0x9C, 0xD2, 0xE9, 0xEF, 0xDD, 0x37, 0x1E, 0x8F, 0xCB, 0x8A, 0x90,
		0xFC, 0x84, 0xE5, 0xF9, 0x14, 0x19, 0xDF, 0x6E, 0x23, 0xC4, 0x66, 0xEB, 0xCC, 0x22, 0x1C, 0x5C,
	}

	decoder := &KODDecoder{
		kodTable: initialKOD,
	}

	// Вычисляем обратную таблицу
	for i, val := range initialKOD {
		decoder.invTable[val] = byte(i)
	}

	return decoder
}

// Decode расшифровывает данные с использованием KOD алгоритма
func (k *KODDecoder) Decode(shift byte, data []byte) []byte {
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = byte((int(k.kodTable[b]) - i - int(shift)) % 256)
	}
	return result
}

// Encode кодирует данные с использованием KOD алгоритма
func (k *KODDecoder) Encode(shift byte, data []byte) []byte {
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = k.invTable[byte((int(b)+i+int(shift))%256)]
	}
	return result
}

// DecodeField декодирует значение поля в зависимости от его типа
func DecodeField(fieldDef FieldDefinition, data []byte) string {
	if len(data) == 0 {
		return ""
	}

	switch fieldDef.Type {
	case FieldTypeSystemNumber:
		// Системный номер - просто преобразуем в строку
		if len(data) >= 4 {
			val := binary.LittleEndian.Uint32(data)
			return strconv.FormatUint(uint64(val), 10)
		}
		return string(data)

	case FieldTypeInteger:
		// Целое число
		if len(data) >= 4 {
			val := binary.LittleEndian.Uint32(data)
			return strconv.FormatUint(uint64(val), 10)
		}
		return strings.TrimRight(string(data), "\x00")

	case FieldTypeString, FieldTypeText:
		// Строка или текст - декодируем из cp1251
		return DecodeCP1251(bytes.TrimRight(data, "\x00"))

	case FieldTypeDate:
		// Дата в формате: <год-1900><месяц:2цифры><день:2цифры>
		dataStr := strings.TrimRight(string(data), "\x00")
		if len(dataStr) >= 6 {
			if year, err := strconv.Atoi(dataStr[:len(dataStr)-4]); err == nil {
				if month, err := strconv.Atoi(dataStr[len(dataStr)-4 : len(dataStr)-2]); err == nil {
					if day, err := strconv.Atoi(dataStr[len(dataStr)-2:]); err == nil {
						date := time.Date(1900+year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
						return date.Format("2006-01-02")
					}
				}
			}
		}
		return dataStr

	case FieldTypeTime:
		// Время в формате: <час:2цифры><минута:2цифры>
		dataStr := strings.TrimRight(string(data), "\x00")
		if len(dataStr) >= 4 {
			if hour, err := strconv.Atoi(dataStr[len(dataStr)-4 : len(dataStr)-2]); err == nil {
				if minute, err := strconv.Atoi(dataStr[len(dataStr)-2:]); err == nil {
					return fmt.Sprintf("%02d:%02d", hour, minute)
				}
			}
		}
		return dataStr

	case FieldTypeFile:
		// Ссылка на файл - извлекаем имя файла
		return DecodeFileReference(data)

	default:
		// Для остальных типов возвращаем как строку
		return DecodeCP1251(bytes.TrimRight(data, "\x00"))
	}
}

// DecodeFileReference декодирует ссылку на файл
func DecodeFileReference(data []byte) string {
	if len(data) < 8 {
		return ""
	}

	reader := bytes.NewReader(data)
	var flag, remLen uint32
	binary.Read(reader, binary.LittleEndian, &flag)
	binary.Read(reader, binary.LittleEndian, &remLen)

	remaining := data[8:]
	parts := bytes.Split(remaining, []byte{0x1e})
	
	if len(parts) >= 3 {
		filename := DecodeCP1251(parts[0])
		extname := DecodeCP1251(parts[1])
		fileRecord := string(parts[2])
		return fmt.Sprintf("%s.%s (%s)", filename, extname, fileRecord)
	}

	return DecodeCP1251(remaining)
}

// DecodeCP1251 декодирует строку из кодировки CP1251 в UTF-8
func DecodeCP1251(data []byte) string {
	// Упрощенная таблица перекодировки CP1251 -> UTF-8
	// Для полной реализации нужно использовать golang.org/x/text/encoding/charmap
	result := make([]rune, 0, len(data))
	
	for _, b := range data {
		if b == 0 {
			break
		}
		
		if b < 128 {
			result = append(result, rune(b))
		} else {
			// Основные русские символы CP1251
			switch b {
			case 0xC0: result = append(result, 'А')
			case 0xC1: result = append(result, 'Б')
			case 0xC2: result = append(result, 'В')
			case 0xC3: result = append(result, 'Г')
			case 0xC4: result = append(result, 'Д')
			case 0xC5: result = append(result, 'Е')
			case 0xC6: result = append(result, 'Ж')
			case 0xC7: result = append(result, 'З')
			case 0xC8: result = append(result, 'И')
			case 0xC9: result = append(result, 'Й')
			case 0xCA: result = append(result, 'К')
			case 0xCB: result = append(result, 'Л')
			case 0xCC: result = append(result, 'М')
			case 0xCD: result = append(result, 'Н')
			case 0xCE: result = append(result, 'О')
			case 0xCF: result = append(result, 'П')
			case 0xD0: result = append(result, 'Р')
			case 0xD1: result = append(result, 'С')
			case 0xD2: result = append(result, 'Т')
			case 0xD3: result = append(result, 'У')
			case 0xD4: result = append(result, 'Ф')
			case 0xD5: result = append(result, 'Х')
			case 0xD6: result = append(result, 'Ц')
			case 0xD7: result = append(result, 'Ч')
			case 0xD8: result = append(result, 'Ш')
			case 0xD9: result = append(result, 'Щ')
			case 0xDA: result = append(result, 'Ъ')
			case 0xDB: result = append(result, 'Ы')
			case 0xDC: result = append(result, 'Ь')
			case 0xDD: result = append(result, 'Э')
			case 0xDE: result = append(result, 'Ю')
			case 0xDF: result = append(result, 'Я')
			case 0xE0: result = append(result, 'а')
			case 0xE1: result = append(result, 'б')
			case 0xE2: result = append(result, 'в')
			case 0xE3: result = append(result, 'г')
			case 0xE4: result = append(result, 'д')
			case 0xE5: result = append(result, 'е')
			case 0xE6: result = append(result, 'ж')
			case 0xE7: result = append(result, 'з')
			case 0xE8: result = append(result, 'и')
			case 0xE9: result = append(result, 'й')
			case 0xEA: result = append(result, 'к')
			case 0xEB: result = append(result, 'л')
			case 0xEC: result = append(result, 'м')
			case 0xED: result = append(result, 'н')
			case 0xEE: result = append(result, 'о')
			case 0xEF: result = append(result, 'п')
			case 0xF0: result = append(result, 'р')
			case 0xF1: result = append(result, 'с')
			case 0xF2: result = append(result, 'т')
			case 0xF3: result = append(result, 'у')
			case 0xF4: result = append(result, 'ф')
			case 0xF5: result = append(result, 'х')
			case 0xF6: result = append(result, 'ц')
			case 0xF7: result = append(result, 'ч')
			case 0xF8: result = append(result, 'ш')
			case 0xF9: result = append(result, 'щ')
			case 0xFA: result = append(result, 'ъ')
			case 0xFB: result = append(result, 'ы')
			case 0xFC: result = append(result, 'ь')
			case 0xFD: result = append(result, 'э')
			case 0xFE: result = append(result, 'ю')
			case 0xFF: result = append(result, 'я')
			default:
				result = append(result, rune(b))
			}
		}
	}
	
	return string(result)
}

// GetClickHouseType возвращает тип ClickHouse для поля
func (f FieldDefinition) GetClickHouseType() string {
	switch f.Type {
	case FieldTypeSystemNumber:
		return "UInt32"
	case FieldTypeInteger:
		return "Int32"
	case FieldTypeString:
		if f.MaxVal > 0 {
			return fmt.Sprintf("String")
		}
		return "String"
	case FieldTypeText:
		return "String"
	case FieldTypeDate:
		return "Date"
	case FieldTypeTime:
		return "String" // Или DateTime если нужно
	case FieldTypeFile:
		return "String"
	default:
		return "String"
	}
}