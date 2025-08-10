package cronodamp

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/mozillazg/go-unidecode"
)

// TransliterateTableName транслитерирует имя таблицы по правилам ClickHouse
// Принимает имя папки и имя таблицы, возвращает валидное имя для ClickHouse
func TransliterateTableName(folderName, tableName string) string {
	// Объединяем имя папки и таблицы
	combined := folderName + "_" + tableName
	
	// Применяем транслитерацию
	transliterated := unidecode.Unidecode(combined)
	
	// Приводим к нижнему регистру
	result := strings.ToLower(transliterated)
	
	// Заменяем недопустимые символы на подчеркивания
	result = sanitizeIdentifier(result)
	
	// Убеждаемся, что имя начинается с буквы или подчеркивания
	if len(result) > 0 && !isValidFirstChar(rune(result[0])) {
		result = "t_" + result
	}
	
	// Ограничиваем длину (ClickHouse поддерживает до 127 символов)
	if len(result) > 127 {
		result = result[:127]
	}
	
	return result
}

// TransliterateColumnName транслитерирует имя колонки по правилам ClickHouse
func TransliterateColumnName(columnName string) string {
	// Применяем транслитерацию
	transliterated := unidecode.Unidecode(columnName)
	
	// Приводим к нижнему регистру
	result := strings.ToLower(transliterated)
	
	// Заменяем недопустимые символы на подчеркивания
	result = sanitizeIdentifier(result)
	
	// Убеждаемся, что имя начинается с буквы или подчеркивания
	if len(result) > 0 && !isValidFirstChar(rune(result[0])) {
		result = "c_" + result
	}
	
	// Ограничиваем длину
	if len(result) > 127 {
		result = result[:127]
	}
	
	// Если результат пустой, используем дефолтное имя
	if result == "" {
		result = "unnamed_column"
	}
	
	return result
}

// sanitizeIdentifier заменяет недопустимые символы на подчеркивания
func sanitizeIdentifier(s string) string {
	// Регулярное выражение для валидных символов в идентификаторах ClickHouse
	// Допустимы: буквы, цифры, подчеркивания
	validChars := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	result := validChars.ReplaceAllString(s, "_")
	
	// Убираем множественные подчеркивания
	multipleUnderscores := regexp.MustCompile(`_+`)
	result = multipleUnderscores.ReplaceAllString(result, "_")
	
	// Убираем подчеркивания в начале и конце
	result = strings.Trim(result, "_")
	
	return result
}

// isValidFirstChar проверяет, может ли символ быть первым в идентификаторе
func isValidFirstChar(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// GetTableName формирует имя таблицы из пути к файлу базы данных
// Извлекает имя папки и имя файла без расширения
func GetTableName(dbFilePath string) string {
	// Убираем расширение файла
	pathWithoutExt := strings.TrimSuffix(dbFilePath, ".db")
	pathWithoutExt = strings.TrimSuffix(pathWithoutExt, ".sqlite")
	pathWithoutExt = strings.TrimSuffix(pathWithoutExt, ".sqlite3")
	
	// Разделяем путь
	parts := strings.Split(pathWithoutExt, "/")
	if len(parts) < 2 {
		// Если нет папки, используем только имя файла
		if len(parts) == 1 {
			return TransliterateTableName("default", parts[0])
		}
		return "default_table"
	}
	
	// Берем последние две части: папку и файл
	folderName := parts[len(parts)-2]
	fileName := parts[len(parts)-1]
	
	return TransliterateTableName(folderName, fileName)
}