package cronodamp

import (
	"testing"
)

func TestTransliterateColumnName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"имя", "imya"},
		{"Фамилия", "familiya"},
		{"ГОРОД", "gorod"},
		{"дата_рождения", "data_rozhdeniya"},
		{"123test", "col_123test"},
		{"test-123", "test_123"},
		{"test@domain.com", "test_domain_com"},
		{"Тест Пробелы", "test_probely"},
		{"", "unknown_column"},
		{"МояДлиннаяКолонкаСОченьДлиннымИменемКотороеПревышаетЛимитВШестьдесятТриСимвола", "moyadlinnayakolonkasochendlinnymimenemkotoroeprevyshaetlimitvsh"},
	}

	for _, test := range tests {
		result := TransliterateColumnName(test.input)
		if result != test.expected {
			t.Errorf("TransliterateColumnName(%q) = %q, ожидалось %q", test.input, result, test.expected)
		}
	}
}

func TestTransliterateTableName(t *testing.T) {
	tests := []struct {
		folder   string
		table    string
		expected string
	}{
		{"отчеты", "продажи", "otchety_prodazhi"},
		{"База данных", "Клиенты 2024", "baza_dannyh_klienty_2024"},
		{"test", "example", "test_example"},
		{"папка/с/подпапками", "таблица", "papka_s_podpapkami_tablitsa"},
		{"", "table", "table"},
		{"folder", "", "folder"},
	}

	for _, test := range tests {
		result := TransliterateTableName(test.folder, test.table)
		if result != test.expected {
			t.Errorf("TransliterateTableName(%q, %q) = %q, ожидалось %q", 
				test.folder, test.table, result, test.expected)
		}
	}
}

func TestIsValidClickHouseName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"valid_name", true},
		{"ValidName", true},
		{"_valid", true},
		{"valid123", true},
		{"123invalid", false},
		{"", false},
		{"test-invalid", false},
		{"test.invalid", false},
		{"test@invalid", false},
		{string(make([]byte, 64)), false}, // слишком длинное имя
	}

	for _, test := range tests {
		result := IsValidClickHouseName(test.name)
		if result != test.expected {
			t.Errorf("IsValidClickHouseName(%q) = %v, ожидалось %v", 
				test.name, result, test.expected)
		}
	}
}

func BenchmarkTransliterateColumnName(b *testing.B) {
	testInput := "ТестоваяКолонкаДляБенчмарка"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TransliterateColumnName(testInput)
	}
}

func BenchmarkTransliterateTableName(b *testing.B) {
	testFolder := "ТестоваяПапка"
	testTable := "ТестоваяТаблица"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TransliterateTableName(testFolder, testTable)
	}
}