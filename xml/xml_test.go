package xml

import (
	"testing"
)

func TestGetPersonalInfo(t *testing.T) {
	// テスト用のXMLデータを定義
	xmlData := []byte(`
		<id extention="hoge">
        <patientPatient>
            <id extention="12345"/>
            <family>Smith</family>
            <birthTime value="2000-01-01"/>
        </patientPatient>
    `)

	// 期待される出力
	expectedEcgID := "hoge"
	expectedPatientID := "12345"
	expectedName := "Smith"
	expectedBirthtime := "2000-01-01"

	// 関数を呼び出し
	ecgID, patientID, name, birthtime, err := GetPersonalInfo(xmlData)

	// エラーチェック
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 結果の確認
	if ecgID != expectedEcgID {
		t.Errorf("unexpected ecgID, got: %s, want: %s", ecgID, expectedEcgID)
	}
	if patientID != expectedPatientID {
		t.Errorf("unexpected patientID, got: %s, want: %s", patientID, expectedPatientID)
	}
	if name != expectedName {
		t.Errorf("unexpected name, got: %s, want: %s", name, expectedName)
	}
	if birthtime != expectedBirthtime {
		t.Errorf("unexpected birthtime, got: %s, want: %s", birthtime, expectedBirthtime)
	}
}
