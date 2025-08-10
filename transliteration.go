package main

import (
	"regexp"
	"strings"
	"unicode"
)

// transliterationMap карта транслитерации русских символов
var transliterationMap = map[rune]string{
	'а': "a", 'А': "A",
	'б': "b", 'Б': "B",
	'в': "v", 'В': "V",
	'г': "g", 'Г': "G",
	'д': "d", 'Д': "D",
	'е': "e", 'Е': "E",
	'ё': "yo", 'Ё': "Yo",
	'ж': "zh", 'Ж': "Zh",
	'з': "z", 'З': "Z",
	'и': "i", 'И': "I",
	'й': "y", 'Й': "Y",
	'к': "k", 'К': "K",
	'л': "l", 'Л': "L",
	'м': "m", 'М': "M",
	'н': "n", 'Н': "N",
	'о': "o", 'О': "O",
	'п': "p", 'П': "P",
	'р': "r", 'Р': "R",
	'с': "s", 'С': "S",
	'т': "t", 'Т': "T",
	'у': "u", 'У': "U",
	'ф': "f", 'Ф': "F",
	'х': "h", 'Х': "H",
	'ц': "ts", 'Ц': "Ts",
	'ч': "ch", 'Ч': "Ch",
	'ш': "sh", 'Ш': "Sh",
	'щ': "sch", 'Щ': "Sch",
	'ъ': "", 'Ъ': "",
	'ы': "y", 'Ы': "Y",
	'ь': "", 'Ь': "",
	'э': "e", 'Э': "E",
	'ю': "yu", 'Ю': "Yu",
	'я': "ya", 'Я': "Ya",
}

// Transliterate транслитерирует русский текст в латиницу
func Transliterate(text string) string {
	var result strings.Builder
	
	for _, r := range text {
		if replacement, exists := transliterationMap[r]; exists {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// SanitizeName очищает имя от недопустимых символов для использования в SQL
func SanitizeName(name string) string {
	// Сначала транслитерируем
	sanitized := Transliterate(name)
	
	// Заменяем недопустимые символы на подчеркивания
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	sanitized = reg.ReplaceAllString(sanitized, "_")
	
	// Убираем множественные подчеркивания
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")
	
	// Убираем подчеркивания в начале и конце
	sanitized = strings.Trim(sanitized, "_")
	
	// Если имя начинается с цифры, добавляем префикс
	if len(sanitized) > 0 && unicode.IsDigit(rune(sanitized[0])) {
		sanitized = "col_" + sanitized
	}
	
	// Если имя пустое, используем дефолтное
	if sanitized == "" {
		sanitized = "unnamed_field"
	}
	
	return strings.ToLower(sanitized)
}

// TransliterateTableName транслитерирует и нормализует имя таблицы
func TransliterateTableName(tableName string) string {
	// Убираем лишние пробелы
	tableName = strings.TrimSpace(tableName)
	
	// Транслитерируем и очищаем
	cleaned := SanitizeName(tableName)
	
	// Добавляем префикс для системных таблиц если нужно
	if strings.HasPrefix(strings.ToUpper(tableName), "RDB$") {
		cleaned = "sys_" + cleaned
	}
	
	return cleaned
}

// TransliterateColumnName транслитерирует и нормализует имя колонки
func TransliterateColumnName(columnName string) string {
	// Убираем лишние пробелы
	columnName = strings.TrimSpace(columnName)
	
	// Транслитерируем и очищаем
	cleaned := SanitizeName(columnName)
	
	// Проверяем зарезервированные слова SQL и добавляем префикс если нужно
	reservedWords := map[string]bool{
		"select": true, "from": true, "where": true, "insert": true,
		"update": true, "delete": true, "create": true, "drop": true,
		"alter": true, "table": true, "index": true, "view": true,
		"database": true, "schema": true, "primary": true, "foreign": true,
		"key": true, "constraint": true, "null": true, "not": true,
		"default": true, "auto_increment": true, "unique": true,
		"order": true, "by": true, "group": true, "having": true,
		"limit": true, "offset": true, "join": true, "inner": true,
		"left": true, "right": true, "outer": true, "union": true,
		"distinct": true, "as": true, "on": true, "using": true,
		"case": true, "when": true, "then": true, "else": true,
		"end": true, "if": true, "exists": true, "in": true,
		"between": true, "like": true, "is": true, "and": true,
		"or": true, "xor": true, "true": true, "false": true,
		"timestamp": true, "datetime": true, "date": true, "time": true,
		"year": true, "month": true, "day": true, "hour": true,
		"minute": true, "second": true, "timezone": true,
	}
	
	if reservedWords[cleaned] {
		cleaned = "col_" + cleaned
	}
	
	return cleaned
}

// ValidateName проверяет, является ли имя допустимым идентификатором SQL
func ValidateName(name string) bool {
	if name == "" {
		return false
	}
	
	// Первый символ должен быть буквой или подчеркиванием
	first := rune(name[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}
	
	// Остальные символы должны быть буквами, цифрами или подчеркиваниями
	for _, r := range name[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	
	return true
}

// NormalizeText нормализует текстовое значение для экспорта
func NormalizeText(text string) string {
	// Убираем управляющие символы
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, text)
	
	// Нормализуем переносы строк
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	
	return cleaned
}