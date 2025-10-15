package transliteration

import (
	"strings"
	"unicode"
)

type Transliterator struct {
	russianToLatin map[rune]string
}

func New() *Transliterator {
	return &Transliterator{
		russianToLatin: map[rune]string{
			'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
			'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
			'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
			'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
			'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
			'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "E",
			'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
			'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
			'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch",
			'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
		},
	}
}

// Transliterate переводит русский текст в транслит для ClickHouse
func (t *Transliterator) Transliterate(text string) string {
	var result strings.Builder
	
	for _, r := range text {
		if latin, exists := t.russianToLatin[r]; exists {
			result.WriteString(latin)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			result.WriteRune(r)
		} else if unicode.IsSpace(r) {
			result.WriteRune('_')
		}
		// Игнорируем все остальные символы
	}
	
	return result.String()
}

// TransliterateFieldName переводит название поля в транслит
func (t *Transliterator) TransliterateFieldName(fieldName string) string {
	transliterated := t.Transliterate(fieldName)
	
	// Убираем множественные подчеркивания
	transliterated = strings.ReplaceAll(transliterated, "__", "_")
	transliterated = strings.Trim(transliterated, "_")
	
	// Если начинается с цифры, добавляем префикс
	if len(transliterated) > 0 && unicode.IsDigit(rune(transliterated[0])) {
		transliterated = "field_" + transliterated
	}
	
	// Если пустая строка, используем дефолтное имя
	if transliterated == "" {
		transliterated = "field"
	}
	
	return strings.ToLower(transliterated)
}

// TransliterateTableName переводит название таблицы в транслит
func (t *Transliterator) TransliterateTableName(tableName string) string {
	transliterated := t.Transliterate(tableName)
	
	// Убираем множественные подчеркивания
	transliterated = strings.ReplaceAll(transliterated, "__", "_")
	transliterated = strings.Trim(transliterated, "_")
	
	// Если начинается с цифры, добавляем префикс
	if len(transliterated) > 0 && unicode.IsDigit(rune(transliterated[0])) {
		transliterated = "table_" + transliterated
	}
	
	// Если пустая строка, используем дефолтное имя
	if transliterated == "" {
		transliterated = "table"
	}
	
	return strings.ToLower(transliterated)
}