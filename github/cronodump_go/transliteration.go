package main

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Transliterator выполняет транслитерацию русских символов в латиницу
type Transliterator struct{}

// NewTransliterator создает новый транслитатор
func NewTransliterator() *Transliterator {
	return &Transliterator{}
}

// Transliterate транслитерирует строку из кириллицы в латиницу
func (t *Transliterator) Transliterate(input string) string {
	// Сначала нормализуем строку
	transformer := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(transformer, input)

	// Словарь транслитерации
	transliterationMap := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
		'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
		'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
		'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
		'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
		'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
		'Ф': "F", 'Х': "Kh", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch",
		'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	var result strings.Builder
	for _, r := range result {
		if replacement, exists := transliterationMap[r]; exists {
			result.WriteString(replacement)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		} else if r == ' ' || r == '_' || r == '-' {
			result.WriteRune('_')
		}
		// Игнорируем остальные символы
	}

	return strings.Trim(result.String(), "_")
}

// TransliterateTableName транслитерирует имя таблицы (папки)
func (t *Transliterator) TransliterateTableName(folderName string) string {
	transliterated := t.Transliterate(folderName)
	// Убираем множественные подчеркивания и делаем валидным именем таблицы
	transliterated = strings.ReplaceAll(transliterated, "__", "_")
	transliterated = strings.Trim(transliterated, "_")
	
	// Если результат пустой, используем fallback
	if transliterated == "" {
		transliterated = "table_" + strings.ReplaceAll(folderName, " ", "_")
	}
	
	return transliterated
}

// TransliterateFieldName транслитерирует имя поля
func (t *Transliterator) TransliterateFieldName(fieldName string) string {
	transliterated := t.Transliterate(fieldName)
	// Убираем множественные подчеркивания
	transliterated = strings.ReplaceAll(transliterated, "__", "_")
	transliterated = strings.Trim(transliterated, "_")
	
	// Если результат пустой, используем fallback
	if transliterated == "" {
		transliterated = "field_" + strings.ReplaceAll(fieldName, " ", "_")
	}
	
	return transliterated
}