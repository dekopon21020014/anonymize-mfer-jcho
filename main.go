package main

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sample/mfer"

	"github.com/rivo/tview"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	wg          sync.WaitGroup
	app         *tview.Application
	list        *tview.List
	currentDir  string
	selectedCSV string
	targetDir   string
	outputFiles []string
)

func main() {
	// ログをファイルに出力する
	logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ログファイル作成失敗: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	defer logFile.Sync()

	app = tview.NewApplication()
	list = tview.NewList()
	list.ShowSecondaryText(false).SetBorder(true).SetTitle(" ファイル / ディレクトリ選択 ")

	currentDir, err = os.UserHomeDir() // os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	showCSVSelection()

	if err := app.SetRoot(list, true).Run(); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}
	wg.Wait()
}

// CSVファイルを選択する画面
func showCSVSelection() {
	list.Clear()
	list.SetTitle(" CSVファイルを選択 ")

	addFilesToList(currentDir, ".csv", func(path string) {
		selectedCSV = path
		readCSVForOutputFiles()
		showDirectorySelection()
	})

	list.AddItem(".. (親ディレクトリへ移動)", "", 0, func() {
		currentDir = filepath.Dir(currentDir)
		showCSVSelection()
	})
}

// 選択したCSVから出力ファイル名を取得 (Shift JIS対応)
func readCSVForOutputFiles() {
	file, err := os.Open(selectedCSV)
	if err != nil {
		log.Fatalf("CSVファイルを開けません: %v", err)
	}
	defer file.Close()

	// Shift JIS → UTF-8 に変換
	reader := csv.NewReader(transform.NewReader(file, japanese.ShiftJIS.NewDecoder()))

	// CSVを読み込む
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("CSVの解析に失敗: %v", err)
	}
	var anonymizedRecords [][]string

	// 取得したファイル名リストを格納
	outputFiles = []string{}
	for _, record := range records {
		if len(record) > 0 {
			outputFiles = append(outputFiles, record[0])

			// 3列目をSHA256に変換
			if len(record) > 2 {
				record[2] = sha256Hash(record[2])
			}

			// 3列目までを表示
			anonymizedRecords = append(anonymizedRecords, record[:3])
		} else {
			log.Println("空行があります")
		}
	}
	log.Println("-------------")

	// ここでanonymizedRecordsをCSVに書き込む
	saveAnonymizedCSV(anonymizedRecords)
}

// 匿名化したデータをCSVに保存
func saveAnonymizedCSV(records [][]string) {
	// 出力ファイル名の作成 (元のファイル名に `_anonymized.csv` を追加)
	outputPath := strings.TrimSuffix(selectedCSV, filepath.Ext(selectedCSV)) + "_anonymized.csv"

	// ファイル作成
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("匿名化CSVの作成に失敗: %v", err)
	}
	defer file.Close()

	// UTF-8 エンコーディングで出力
	writer := csv.NewWriter(file)

	// CSV にデータを書き込む
	err = writer.WriteAll(records)
	if err != nil {
		log.Fatalf("CSVの書き込みに失敗: %v", err)
	}

	// 書き込みを確実にフラッシュ
	writer.Flush()

	log.Printf("匿名化CSVを出力しました: %s", outputPath)
}

// 探索するディレクトリを選択
func showDirectorySelection() {
	list.Clear()
	list.SetTitle(" 出力ファイル探索ディレクトリを選択 ")

	addDirsToList(currentDir, func(path string) {
		targetDir = path
		processFiles()
	})

	list.AddItem(".. (親ディレクトリへ移動)", "", 0, func() {
		currentDir = filepath.Dir(currentDir)
		showDirectorySelection()
	})
}

// ファイル処理を実行
func processFiles() {
	log.Printf("探索ディレクトリ: %s", targetDir)

	for _, filename := range outputFiles {
		log.Println("処理対象ファイル: " + filename)
		filePath := searchFile(targetDir, filename)
		if filePath == "" {
			log.Printf("ファイルが見つかりません: %s", filename)
			continue
		}

		if strings.HasSuffix(strings.ToLower(filePath), ".mwf") {
			wg.Add(1) // ゴルーチンを開始する前にカウント
			go processFile(filePath)
		}
	}

	// すべての処理が終わったら TUI を終了
	wg.Wait()
	app.Stop()
}

// ファイルを検索する
func searchFile(root, filename string) string {
	var foundPath string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && filepath.Base(path) == filename {
			foundPath = path
		}
		return nil
	})
	return foundPath
}

// ファイルを処理する
func processFile(filePath string) {
	defer wg.Done() // 終了時にカウントを減らす

	log.Printf("処理中: %s", filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("読み込み失敗: %v", err)
		return
	}
	anonimizedData, err := mfer.Anonymize(data)
	if err != nil {
		log.Printf("処理失敗: %v", err)
		return
	}

	if err := os.WriteFile("anonymized_"+filepath.Base(filePath), anonimizedData, 0666); err != nil {
		log.Printf("書き込み失敗: %v", err)
		return
	}
	log.Printf("処理完了: %s", filePath)
}

// 指定された拡張子のファイルをリストに追加
func addFilesToList(dir, ext string, callback func(string)) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("ディレクトリの読み取り失敗: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ext) {
			filePath := filepath.Join(dir, entry.Name())
			list.AddItem(entry.Name(), "", 0, func() {
				callback(filePath)
			})
		}
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirPath := filepath.Join(dir, entry.Name())
			list.AddItem("[DIR] "+entry.Name(), "", 0, func() {
				currentDir = dirPath
				showCSVSelection()
			})
		}
	}
}

// ディレクトリをリストに追加
func addDirsToList(dir string, callback func(string)) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("ディレクトリの読み取り失敗: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(dir, entry.Name())
			list.AddItem(entry.Name(), "", 0, func() {
				callback(dirPath)
			})
		}
	}
}

// SHA256 ハッシュ関数
func sha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
