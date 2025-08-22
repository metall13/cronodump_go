package main

import (
	"encoding/binary"
	"fmt"
)

// readTADHeader читает заголовок .tad файла
func (cf *CronosFile) readTADHeader() error {
	_, err := cf.tadFile.Seek(0, 0)
	if err != nil {
		return err
	}

	if cf.isV3() {
		// V3 формат: 2 uint32
		var nrDeleted, firstDeleted uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &nrDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &firstDeleted); err != nil {
			return err
		}
		cf.tadHdrLen = 8
	} else if cf.isV4() {
		// V4 формат: 4 uint32
		var unk1, nrDeleted, firstDeleted, unk2 uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &unk1); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &nrDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &firstDeleted); err != nil {
			return err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &unk2); err != nil {
			return err
		}
		cf.tadHdrLen = 16
	} else {
		return fmt.Errorf("неподдерживаемая версия TAD файла")
	}

	// Определяем размер записи в TAD
	if cf.use64bit() {
		cf.tadEntrySize = 16 // 8 bytes offset + 4 bytes size + 4 bytes checksum
	} else {
		cf.tadEntrySize = 12 // 4 bytes offset + 4 bytes size + 4 bytes checksum
	}

	// Вычисляем количество записей
	tadFileSize, err := cf.tadFile.Seek(0, 2) // Переходим в конец файла
	if err != nil {
		return err
	}

	tadDataSize := int(tadFileSize) - cf.tadHdrLen
	cf.nrOfRecords = tadDataSize / cf.tadEntrySize

	return nil
}

// use64bit проверяет использует ли файл 64-битные смещения
func (cf *CronosFile) use64bit() bool {
	version := string(cf.header.Version[:])
	return version == "01.03" || version == "01.05" || version == "01.11"
}

// isV3 проверяет версию 3
func (cf *CronosFile) isV3() bool {
	version := string(cf.header.Version[:])
	return version == "01.02" || version == "01.03" || version == "01.04" || version == "01.05"
}

// isV4 проверяет версию 4
func (cf *CronosFile) isV4() bool {
	version := string(cf.header.Version[:])
	return version == "01.11" || version == "01.13" || version == "01.14"
}

// isEncoded проверяет закодированы ли данные
func (cf *CronosFile) isEncoded() bool {
	return cf.header.Encoding&1 != 0
}

// readRecord читает запись из файла
func (cf *CronosFile) readRecord(recordID int) ([]byte, error) {
	// Читаем индекс записи из .tad файла
	tadOffset := int64(cf.tadHdrLen + (recordID-1)*cf.tadEntrySize)
	
	_, err := cf.tadFile.Seek(tadOffset, 0)
	if err != nil {
		return nil, err
	}

	var offset uint64
	var size uint32
	var checksum uint32

	if cf.use64bit() {
		// 64-битные смещения: offset(8) + size(4) + checksum(4)
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &offset); err != nil {
			return nil, err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
	} else {
		// 32-битные смещения: offset(4) + size(4) + checksum(4)
		var offset32 uint32
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &offset32); err != nil {
			return nil, err
		}
		offset = uint64(offset32)
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &size); err != nil {
			return nil, err
		}
		if err := binary.Read(cf.tadFile, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
	}

	// Проверяем на удаленную запись
	if size == 0xFFFFFFFF {
		return nil, fmt.Errorf("запись удалена")
	}

	// Обрабатываем флаги в зависимости от версии
	var flags uint32
	var actualSize uint32
	if cf.isV3() {
		flags = size >> 24
		actualSize = size & 0x0FFFFFFF
	} else if cf.isV4() {
		flags = uint32(offset >> 56)
		offset = offset & ((1 << 56) - 1)
		actualSize = size
	} else {
		actualSize = size
	}

	// Читаем данные из .dat файла
	_, err = cf.datFile.Seek(int64(offset), 0)
	if err != nil {
		return nil, err
	}

	data := make([]byte, actualSize)
	_, err = cf.datFile.Read(data)
	if err != nil {
		return nil, err
	}

	// Обрабатываем расширенные записи (flags == 0)
	if flags == 0 && len(data) > 0 {
		var extOffset uint64
		var extLen uint32
		if cf.use64bit() {
			extOffset = binary.LittleEndian.Uint64(data[:8])
			extLen = binary.LittleEndian.Uint32(data[8:12])
			data = data[12:]
		} else {
			extOffset = uint64(binary.LittleEndian.Uint32(data[:4]))
			extLen = binary.LittleEndian.Uint32(data[4:8])
			data = data[8:]
		}

		// Читаем дополнительные данные если нужно
		for uint32(len(data)) < extLen {
			additionalData := make([]byte, cf.header.BlockSize)
			_, err = cf.datFile.Seek(int64(extOffset), 0)
			if err != nil {
				break
			}
			_, err = cf.datFile.Read(additionalData)
			if err != nil {
				break
			}

			if cf.use64bit() {
				extOffset = binary.LittleEndian.Uint64(additionalData[:8])
				data = append(data, additionalData[8:]...)
			} else {
				extOffset = uint64(binary.LittleEndian.Uint32(additionalData[:4]))
				data = append(data, additionalData[4:]...)
			}
		}

		// Обрезаем до нужной длины
		if uint32(len(data)) > extLen {
			data = data[:extLen]
		}
	}

	// Декодируем с помощью KOD если нужно
	if cf.isEncoded() && cf.kod != nil {
		data = cf.kod.Decode(recordID, data)
	}

	return data, nil
}

// decodeDBDefinition декодирует определения базы данных
func (p *CronosParser) decodeDBDefinition(data []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	rd := NewByteReader(data)

	for !rd.EOF() {
		keyName, err := rd.ReadName()
		if err != nil {
			break
		}

		indexOrLength, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}

		if indexOrLength&0x80000000 != 0 {
			// Прямые данные
			length := indexOrLength & 0x7FFFFFFF
			value, err := rd.ReadBytes(int(length))
			if err != nil {
				return nil, err
			}
			result[keyName] = value
		} else {
			// Ссылка на другую запись
			refData, err := p.stru.readRecord(int(indexOrLength))
			if err != nil {
				return nil, err
			}
			if len(refData) > 0 && refData[0] == 0x04 {
				result[keyName] = refData[1:]
			} else {
				result[keyName] = refData
			}
		}
	}

	return result, nil
}

// parseTableDefinition парсит определение таблицы
func (p *CronosParser) parseTableDefinition(data, imageData []byte) (*Table, error) {
	rd := NewByteReader(data)

	// Читаем заголовок таблицы
	unk1, err := rd.ReadWord()
	if err != nil {
		return nil, err
	}
	_ = unk1

	version, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	if version > 1 {
		_, err := rd.ReadByte() // Пропускаем байт
		if err != nil {
			return nil, err
		}
	}

	unk2, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}

	unk3, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}
	_ = unk3

	if unk2 > 5 {
		_, err := rd.ReadDWord() // Дополнительные данные
		if err != nil {
			return nil, err
		}
	}

	unk4, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}
	_ = unk4

	tableID, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	tableName, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	abbrev, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	unk7, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}
	_ = unk7

	nrFields, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	// Читаем определения полей
	var fields []Field
	for i := 0; i < int(nrFields); i++ {
		defLen, err := rd.ReadWord()
		if err != nil {
			return nil, err
		}

		fieldDefData, err := rd.ReadBytes(int(defLen))
		if err != nil {
			return nil, err
		}

		field, err := p.parseFieldDefinition(fieldDefData)
		if err != nil {
			if p.verbose {
				fmt.Printf("Ошибка парсинга поля %d: %v\n", i, err)
			}
			continue
		}

		fields = append(fields, *field)
	}

	table := &Table{
		ID:     tableID,
		Name:   tableName,
		Abbrev: abbrev,
		Fields: fields,
	}

	return table, nil
}

// parseFieldDefinition парсит определение поля
func (p *CronosParser) parseFieldDefinition(data []byte) (*Field, error) {
	rd := NewByteReader(data)

	typ, err := rd.ReadWord()
	if err != nil {
		return nil, err
	}

	idx1, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}
	_ = idx1

	name, err := rd.ReadName()
	if err != nil {
		return nil, err
	}

	flags, err := rd.ReadDWord()
	if err != nil {
		return nil, err
	}

	minVal, err := rd.ReadByte()
	if err != nil {
		return nil, err
	}
	_ = minVal

	var maxLen uint32
	if typ != 0 {
		idx2, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}
		_ = idx2 // Пока не используем

		maxLen, err = rd.ReadDWord()
		if err != nil {
			return nil, err
		}

		unk4, err := rd.ReadDWord()
		if err != nil {
			return nil, err
		}
		_ = unk4 // Пока не используем
	}

	field := &Field{
		Name:   name,
		Type:   int(typ),
		MaxLen: maxLen,
		Flags:  flags,
	}

	return field, nil
}

// parseRecord парсит запись таблицы
func (p *CronosParser) parseRecord(table *Table, data []byte) (*Record, error) {
	rd := NewByteReader(data)
	var fieldValues []FieldValue

	for _, field := range table.Fields {
		value, err := p.parseFieldValue(field, rd)
		if err != nil {
			// Если не удалось прочитать поле, добавляем пустое значение
			fieldValues = append(fieldValues, FieldValue{
				Type:    field.Type,
				Content: "",
			})
			continue
		}

		fieldValues = append(fieldValues, *value)
	}

	return &Record{Fields: fieldValues}, nil
}

// parseFieldValue парсит значение поля
func (p *CronosParser) parseFieldValue(field Field, rd *ByteReader) (*FieldValue, error) {
	var content string

	switch field.Type {
	case 0: // Системный номер (ID)
		// Читаем как число
		if !rd.EOF() {
			data, err := rd.ReadBytes(4)
			if err == nil && len(data) == 4 {
				value := binary.LittleEndian.Uint32(data)
				content = fmt.Sprintf("%d", value)
			}
		}

	case 1: // Целое число
		if !rd.EOF() {
			data, err := rd.ReadBytes(4)
			if err == nil && len(data) == 4 {
				value := binary.LittleEndian.Uint32(data)
				content = fmt.Sprintf("%d", value)
			}
		}

	case 2: // Строка (VARCHAR)
		if !rd.EOF() {
			str, err := rd.ReadName()
			if err == nil {
				content = str
			}
		}

	case 3: // Текст (словарь)
		if !rd.EOF() {
			str, err := rd.ReadLongString()
			if err == nil {
				content = str
			}
		}

	case 4: // Дата
		if !rd.EOF() {
			dateBytes, err := rd.ReadBytes(6) // Формат: YYMMDD
			if err == nil && len(dateBytes) >= 6 {
				// Парсим дату в формате YYMMDD
				year := 1900 + int(dateBytes[0]) + int(dateBytes[1])*256
				month := int(dateBytes[2]) + int(dateBytes[3])*256
				day := int(dateBytes[4]) + int(dateBytes[5])*256
				content = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			}
		}

	case 5: // Время
		if !rd.EOF() {
			timeBytes, err := rd.ReadBytes(4) // Формат: HHMM
			if err == nil && len(timeBytes) >= 4 {
				hour := int(timeBytes[0]) + int(timeBytes[1])*256
				minute := int(timeBytes[2]) + int(timeBytes[3])*256
				content = fmt.Sprintf("%02d:%02d:00", hour, minute)
			}
		}

	case 6: // Ссылка на файл
		if !rd.EOF() {
			str, err := rd.ReadName()
			if err == nil {
				content = str
			}
		}

	default:
		// Неизвестный тип, читаем как строку
		if !rd.EOF() {
			remaining := rd.Remaining()
			content = string(remaining)
		}
	}

	return &FieldValue{
		Type:    field.Type,
		Content: content,
	}, nil
}