package main

import (
	"regexp"
	"strings"
	"unicode"
)

// TransliterationMap таблица транслитерации русских букв
var TransliterationMap = map[rune]string{
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

// Transliterate переводит строку в латиницу
func Transliterate(input string) string {
	var result strings.Builder
	
	for _, char := range input {
		if transliterated, exists := TransliterationMap[char]; exists {
			result.WriteString(transliterated)
		} else {
			result.WriteRune(char)
		}
	}
	
	return result.String()
}

// SanitizeForClickHouse приводит имя к формату, подходящему для ClickHouse
func SanitizeForClickHouse(name string) string {
	// Сначала транслитерируем
	name = Transliterate(name)
	
	// Удаляем или заменяем недопустимые символы
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, "{", "_")
	name = strings.ReplaceAll(name, "}", "_")
	name = strings.ReplaceAll(name, "@", "_")
	name = strings.ReplaceAll(name, "#", "_")
	name = strings.ReplaceAll(name, "$", "_")
	name = strings.ReplaceAll(name, "%", "_")
	name = strings.ReplaceAll(name, "^", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "+", "_")
	name = strings.ReplaceAll(name, "=", "_")
	name = strings.ReplaceAll(name, "!", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, ";", "_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "'", "_")
	name = strings.ReplaceAll(name, "`", "_")
	
	// Удаляем множественные подчеркивания
	re := regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")
	
	// Убираем подчеркивания в начале и конце
	name = strings.Trim(name, "_")
	
	// Если имя пустое, используем fallback
	if name == "" {
		name = "unnamed"
	}
	
	// Если имя начинается с цифры, добавляем префикс
	if len(name) > 0 && unicode.IsDigit(rune(name[0])) {
		name = "t_" + name
	}
	
	// Приводим к нижнему регистру для консистентности
	name = strings.ToLower(name)
	
	return name
}

// TransliterateTableName создает корректное имя таблицы на основе папки и имени таблицы
func TransliterateTableName(folderName, tableName string) string {
	folder := SanitizeForClickHouse(folderName)
	table := SanitizeForClickHouse(tableName)
	
	// Комбинируем имя папки и таблицы
	if folder != "" && folder != "unnamed" && table != "" && table != "unnamed" {
		return folder + "_" + table
	} else if folder != "" && folder != "unnamed" {
		return folder
	} else if table != "" && table != "unnamed" {
		return table
	}
	
	return "unknown_table"
}

// TransliterateColumnName создает корректное имя колонки
func TransliterateColumnName(columnName string) string {
	name := SanitizeForClickHouse(columnName)
	
	// Проверяем зарезервированные слова ClickHouse
	reservedWords := map[string]bool{
		"select": true, "from": true, "where": true, "group": true, "by": true,
		"order": true, "having": true, "limit": true, "offset": true, "union": true,
		"all": true, "distinct": true, "as": true, "on": true, "using": true,
		"join": true, "left": true, "right": true, "inner": true, "outer": true,
		"full": true, "cross": true, "natural": true, "with": true, "recursive": true,
		"case": true, "when": true, "then": true, "else": true, "end": true,
		"if": true, "exists": true, "not": true, "null": true, "true": true, "false": true,
		"and": true, "or": true, "between": true, "like": true, "in": true,
		"is": true, "any": true, "some": true, "array": true,
	}
	
	// Проверяем точное совпадение (учитывая регистр)
	if reservedWords[strings.ToLower(name)] {
		name = "col_" + name
	}
	
	return name
}