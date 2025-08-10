package main

import (
	"regexp"
	"strings"
	"unicode"
)

// transliterationMap содержит карту транслитерации русских букв в латиницу
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
	// Украинские буквы
	'є': "ye", 'Є': "Ye",
	'і': "i", 'І': "I",
	'ї': "yi", 'Ї': "Yi",
	'ґ': "g", 'Ґ': "G",
	// Белорусские буквы
	'ў': "u", 'Ў': "U",
}

// Регулярные выражения для валидации имен ClickHouse
var (
	// ClickHouse identifier должен начинаться с буквы или подчеркивания
	clickHouseNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// Недопустимые символы для замены
	invalidCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	// Множественные подчеркивания
	multipleUnderscoresRegex = regexp.MustCompile(`_{2,}`)
)

// Transliterate выполняет транслитерацию текста с кириллицы на латиницу
func Transliterate(text string) string {
	var result strings.Builder
	
	for _, r := range text {
		if replacement, exists := transliterationMap[r]; exists {
			result.WriteString(replacement)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else {
			// Заменяем все остальные символы на подчеркивание
			result.WriteString("_")
		}
	}
	
	return result.String()
}

// SanitizeClickHouseName приводит имя к требованиям ClickHouse
func SanitizeClickHouseName(name string) string {
	if name == "" {
		return "unnamed"
	}
	
	// Транслитерируем
	transliterated := Transliterate(name)
	
	// Заменяем недопустимые символы на подчеркивания
	sanitized := invalidCharsRegex.ReplaceAllString(transliterated, "_")
	
	// Убираем множественные подчеркивания
	sanitized = multipleUnderscoresRegex.ReplaceAllString(sanitized, "_")
	
	// Убираем подчеркивания в начале и конце
	sanitized = strings.Trim(sanitized, "_")
	
	// Если имя пустое после очистки
	if sanitized == "" {
		sanitized = "unnamed"
	}
	
	// Если имя начинается с цифры, добавляем префикс
	if len(sanitized) > 0 && unicode.IsDigit(rune(sanitized[0])) {
		sanitized = "col_" + sanitized
	}
	
	// Проверяем, не является ли имя зарезервированным словом ClickHouse
	sanitized = avoidReservedWords(sanitized)
	
	return sanitized
}

// TransliterateTableName создает имя таблицы из имени папки и имени таблицы
func TransliterateTableName(folderName, tableName string) string {
	// Очищаем имена
	cleanFolderName := SanitizeClickHouseName(folderName)
	cleanTableName := SanitizeClickHouseName(tableName)
	
	// Объединяем имена
	var fullName string
	if cleanFolderName != "" && cleanTableName != "" {
		fullName = cleanFolderName + "_" + cleanTableName
	} else if cleanFolderName != "" {
		fullName = cleanFolderName
	} else {
		fullName = cleanTableName
	}
	
	// Дополнительная очистка
	result := SanitizeClickHouseName(fullName)
	
	// Ограничиваем длину имени (ClickHouse имеет ограничения)
	maxLength := 127
	if len(result) > maxLength {
		result = result[:maxLength]
		// Убираем подчеркивание в конце, если оно есть
		result = strings.TrimRight(result, "_")
	}
	
	return result
}

// TransliterateColumnName создает имя колонки
func TransliterateColumnName(columnName string) string {
	result := SanitizeClickHouseName(columnName)
	
	// Ограничиваем длину имени колонки
	maxLength := 127
	if len(result) > maxLength {
		result = result[:maxLength]
		result = strings.TrimRight(result, "_")
	}
	
	return result
}

// avoidReservedWords проверяет и избегает зарезервированных слов ClickHouse
func avoidReservedWords(name string) string {
	// Список основных зарезервированных слов ClickHouse
	reservedWords := map[string]bool{
		"select": true, "insert": true, "update": true, "delete": true,
		"create": true, "drop": true, "alter": true, "table": true,
		"database": true, "index": true, "view": true, "trigger": true,
		"procedure": true, "function": true, "from": true, "where": true,
		"group": true, "order": true, "having": true, "limit": true,
		"offset": true, "union": true, "join": true, "inner": true,
		"left": true, "right": true, "full": true, "outer": true,
		"on": true, "as": true, "and": true, "or": true, "not": true,
		"null": true, "true": true, "false": true, "case": true,
		"when": true, "then": true, "else": true, "end": true,
		"if": true, "exists": true, "between": true, "like": true,
		"in": true, "is": true, "distinct": true, "all": true,
		"any": true, "some": true, "primary": true,
		"key": true, "foreign": true, "references": true, "check": true,
		"constraint": true, "default": true, "auto_increment": true,
		"engine": true, "partition": true, "settings": true,
	}
	
	lowerName := strings.ToLower(name)
	if reservedWords[lowerName] {
		return name + "_col"
	}
	
	return name
}

// ValidateClickHouseName проверяет, соответствует ли имя требованиям ClickHouse
func ValidateClickHouseName(name string) bool {
	if name == "" {
		return false
	}
	
	// Проверяем регулярным выражением
	if !clickHouseNameRegex.MatchString(name) {
		return false
	}
	
	// Проверяем длину
	if len(name) > 127 {
		return false
	}
	
	return true
}