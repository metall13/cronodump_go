package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TransliterationMap карта транслитерации русских символов
var TransliterationMap = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E", 'Ж': "ZH",
	'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O",
	'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "KH", 'Ц': "TS",
	'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "YU", 'Я': "YA",
	
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh",
	'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts",
	'ч': "ch", 'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// Transliterate транслитерирует русский текст в латиницу
func Transliterate(text string) string {
	var result strings.Builder
	
	for _, r := range text {
		if trans, exists := TransliterationMap[r]; exists {
			result.WriteString(trans)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' {
			result.WriteString("_")
		}
		// Пропускаем все остальные символы
	}
	
	return result.String()
}

// TransliterateTableName транслитерирует имя таблицы согласно правилам ClickHouse
func TransliterateTableName(folderName, tableName string) string {
	// Объединяем имя папки и имя таблицы
	fullName := fmt.Sprintf("%s_%s", folderName, tableName)
	
	// Транслитерируем
	transliterated := Transliterate(fullName)
	
	// Приводим к нижнему регистру
	transliterated = strings.ToLower(transliterated)
	
	// Убираем множественные подчеркивания
	re := regexp.MustCompile(`_+`)
	transliterated = re.ReplaceAllString(transliterated, "_")
	
	// Убираем подчеркивания в начале и конце
	transliterated = strings.Trim(transliterated, "_")
	
	// Проверяем что имя начинается с буквы или подчеркивания
	if len(transliterated) > 0 && !unicode.IsLetter(rune(transliterated[0])) && transliterated[0] != '_' {
		transliterated = "t_" + transliterated
	}
	
	// Если имя пустое, задаем дефолтное
	if transliterated == "" {
		transliterated = "unknown_table"
	}
	
	return transliterated
}

// TransliterateColumnName транслитерирует имя колонки согласно правилам ClickHouse
func TransliterateColumnName(columnName string) string {
	// Транслитерируем
	transliterated := Transliterate(columnName)
	
	// Приводим к нижнему регистру
	transliterated = strings.ToLower(transliterated)
	
	// Заменяем пробелы и спецсимволы на подчеркивания
	re := regexp.MustCompile(`[^a-z0-9_]`)
	transliterated = re.ReplaceAllString(transliterated, "_")
	
	// Убираем множественные подчеркивания
	re = regexp.MustCompile(`_+`)
	transliterated = re.ReplaceAllString(transliterated, "_")
	
	// Убираем подчеркивания в начале и конце
	transliterated = strings.Trim(transliterated, "_")
	
	// Проверяем что имя начинается с буквы или подчеркивания
	if len(transliterated) > 0 && !unicode.IsLetter(rune(transliterated[0])) && transliterated[0] != '_' {
		transliterated = "col_" + transliterated
	}
	
	// Если имя пустое, задаем дефолтное
	if transliterated == "" {
		transliterated = "unknown_column"
	}
	
	// Проверяем зарезервированные слова ClickHouse
	reservedWords := map[string]bool{
		"select": true, "from": true, "where": true, "group": true, "order": true,
		"by": true, "having": true, "limit": true, "offset": true, "union": true,
		"all": true, "distinct": true, "as": true, "and": true, "or": true, "not": true,
		"in": true, "exists": true, "between": true, "like": true, "is": true, "null": true,
		"true": true, "false": true, "case": true, "when": true, "then": true, "else": true,
		"end": true, "if": true, "create": true, "table": true, "database": true, "drop": true,
		"alter": true, "insert": true, "into": true, "values": true, "update": true, "set": true,
		"delete": true, "truncate": true, "index": true, "key": true, "primary": true,
		"foreign": true, "references": true, "constraint": true, "unique": true, "check": true,
		"default": true, "auto_increment": true, "comment": true,
	}
	
	if reservedWords[transliterated] {
		transliterated = "col_" + transliterated
	}
	
	return transliterated
}

// ExtractFolderName извлекает имя папки из пути к базе данных
func ExtractFolderName(dbPath string) string {
	// Получаем имя папки из пути
	parts := strings.Split(strings.ReplaceAll(dbPath, "\\", "/"), "/")
	
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	
	return "unknown_folder"
}