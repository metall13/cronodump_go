package cronodamp

import (
	"regexp"
	"strings"
	"unicode"
)

// translitMap карта транслитерации русских символов
var translitMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
	
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
	'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
	'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
	'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch",
	'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
}

// TransliterateColumnName транслитерирует имя колонки для ClickHouse
func TransliterateColumnName(name string) string {
	return sanitizeClickHouseName(transliterate(name))
}

// TransliterateTableName транслитерирует имя таблицы для ClickHouse
func TransliterateTableName(folderName, tableName string) string {
	// Объединяем имя папки и таблицы через подчеркивание
	fullName := folderName + "_" + tableName
	return sanitizeClickHouseName(transliterate(fullName))
}

// transliterate выполняет транслитерацию строки
func transliterate(input string) string {
	var result strings.Builder
	
	for _, char := range input {
		if translitChar, ok := translitMap[char]; ok {
			result.WriteString(translitChar)
		} else if unicode.IsLetter(char) || unicode.IsDigit(char) {
			result.WriteRune(char)
		} else {
			// Заменяем все небуквенные символы на подчеркивание
			result.WriteString("_")
		}
	}
	
	return result.String()
}

// sanitizeClickHouseName приводит имя к требованиям ClickHouse
func sanitizeClickHouseName(name string) string {
	// Удаляем множественные подчеркивания
	re := regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")
	
	// Удаляем подчеркивания в начале и конце
	name = strings.Trim(name, "_")
	
	// Если имя пустое, используем дефолтное
	if name == "" {
		name = "unknown_column"
	}
	
	// Если имя начинается с цифры, добавляем префикс
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "col_" + name
	}
	
	// ClickHouse имена не должны быть длиннее 63 символов
	if len(name) > 63 {
		name = name[:63]
	}
	
	return strings.ToLower(name)
}

// IsValidClickHouseName проверяет, является ли имя валидным для ClickHouse
func IsValidClickHouseName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	
	// Должно начинаться с буквы или подчеркивания
	if !unicode.IsLetter(rune(name[0])) && name[0] != '_' {
		return false
	}
	
	// Может содержать только буквы, цифры и подчеркивания
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return false
		}
	}
	
	return true
}