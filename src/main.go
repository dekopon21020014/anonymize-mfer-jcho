package main

import (
	"anonymize-mfer-jcho/mfer"
	"anonymize-mfer-jcho/table"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rivo/tview"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	wg            sync.WaitGroup
	app           *tview.Application
	layout        *tview.Flex
	password      string
	saveDir       string
	currentDir    string
	selectedCSV   string
	targetDir     string
	outputFiles   []string
	patientsTable *table.PatientsTable

	// 各種UI
	passwordForm *tview.Form
	saveDirList  *tview.List
	csvList      *tview.List
	mwfDirList   *tview.List
	logView      *tview.TextView
)

// ** メイン関数 **
func main() {
	// .envファイルを読み込む
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	patientsTable = table.SetupDatabase()
	defer patientsTable.DB.Close()

	setupLogger()

	for { // ユーザが"終了"を選択するまでループ
		initializeTUI()
		wg = sync.WaitGroup{}
		if err := app.Run(); err != nil {
			log.Fatalf("failed to start app: %v", err)
		}
		wg.Wait()
	}
}

// ** ログ設定 **
func setupLogger() {
	logFile, err := os.OpenFile(
		os.Getenv("LOG_FILE_DIR")+os.Getenv("LOG_FILE_NAME"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("ログファイル作成失敗: %v", err)
	}
	log.SetOutput(logFile)
}

// ** TUIの初期化 **
func initializeTUI() {
	app = tview.NewApplication()
	// currentDir, _ = os.Getwd()
	currentDir = os.Getenv("CURRENT_DIR")

	// 各画面を作成
	passwordForm = createPasswordForm()
	saveDirList = createSaveDirList()
	csvList = createCSVList()
	mwfDirList = createMWFDirectoryList()
	logView = createLogView()

	// **最初はパスワード画面を表示**
	layout = tview.NewFlex().
		AddItem(passwordForm, 0, 1, true).
		AddItem(saveDirList, 0, 1, false).
		AddItem(csvList, 0, 1, false).
		AddItem(mwfDirList, 0, 1, false)

	app.SetRoot(layout, true)
}

// ** パスワード入力フォーム **
func createPasswordForm() *tview.Form {
	form := tview.NewForm().
		AddPasswordField("パスワード:", "", 20, '*', func(text string) {
			password = text
		}).
		AddButton("次へ", func() {
			updateSaveDirList()
		}).
		AddButton("終了", func() {
			app.Stop()
			os.Exit(0)
		})

	form.SetBorder(true).SetTitle("1. パスワード入力")
	return form
}

// ** 保存先ディレクトリ選択 **
func createSaveDirList() *tview.List {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("2. 保存先フォルダを選択")
	return list
}

// ** CSV選択リスト **
func createCSVList() *tview.List {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("3. CSVファイル選択")
	return list
}

// ** MWFフォルダ選択リスト **
func createMWFDirectoryList() *tview.List {
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("4. MWFディレクトリ選択")
	return list
}

// ** ログ画面 **
func createLogView() *tview.TextView {
	logView := tview.NewTextView().SetDynamicColors(true)
	logView.SetBorder(true).SetTitle("ログ")
	return logView
}

// **保存フォルダリストを更新**
func updateSaveDirList() {
	go func() { // 非同期で処理
		saveDirList.Clear()
		app.QueueUpdateDraw(func() {
			saveDirList.SetTitle("保存先フォルダ: " + filepath.Base(currentDir))
		})

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			app.QueueUpdateDraw(func() {
				logView.SetText("ディレクトリ読み取り失敗: " + err.Error())
			})
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				dirPath := filepath.Join(currentDir, entry.Name())
				app.QueueUpdateDraw(func() {
					saveDirList.AddItem(entry.Name(), "", 0, func() {
						currentDir = dirPath
						updateSaveDirList()
					})
				})
			}
		}

		// 親ディレクトリ (..) を追加
		parentDir := filepath.Dir(currentDir)
		saveDirList.AddItem("[DIR] 前のフォルダに戻る", "", 0, func() {
			currentDir = parentDir
			updateSaveDirList()
		})

		// 保存先フォルダを選択
		saveDirList.AddItem("[✔] このフォルダを選択", "", 0, func() {
			saveDir = currentDir
			updateCSVList()
		})

		// **UIを更新**
		app.QueueUpdateDraw(func() {
			app.SetFocus(saveDirList)
		})
	}()
}

// ** [修正] CSV選択リストを更新 **
func updateCSVList() {
	go func() { // 非同期で処理
		csvList.Clear()

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			logView.SetText("ディレクトリ読み取り失敗: " + err.Error())
			return
		}

		for _, entry := range entries {
			filePath := filepath.Join(currentDir, entry.Name())

			if entry.IsDir() {
				csvList.AddItem("[DIR] "+entry.Name(), "", 0, func() {
					currentDir = filePath
					updateCSVList()
				})
			} else if strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
				csvList.AddItem(entry.Name(), "", 0, func() {
					selectedCSV = filePath
					readCSVForOutputFiles()
					updateMWFDirectoryList()
				})
			}
		}

		// 親ディレクトリ (..) を追加
		parentDir := filepath.Dir(currentDir)
		csvList.AddItem("[DIR] 前のフォルダに戻る", "", 0, func() {
			currentDir = parentDir
			updateCSVList()
		})

		// **[追加] 画面を更新 & フォーカス移動**
		app.SetFocus(csvList)
		app.Draw()
	}()
}

// ** MWFフォルダリストを更新 **
func updateMWFDirectoryList() {
	go func() { // 非同期で処理
		mwfDirList.Clear()

		// **[現在のフォルダをタイトルに表示]**
		app.QueueUpdateDraw(func() {
			mwfDirList.SetTitle("MWFフォルダ選択: " + filepath.Base(currentDir))
		})

		entries, err := os.ReadDir(currentDir)
		if err != nil {
			logView.SetText("ディレクトリ読み取り失敗: " + err.Error())
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				dirPath := filepath.Join(currentDir, entry.Name())
				mwfDirList.AddItem("[DIR] "+entry.Name(), "", 0, func() {
					currentDir = dirPath
					updateMWFDirectoryList()
				})
			}
		}

		parentDir := filepath.Dir(currentDir)
		mwfDirList.AddItem("[DIR] 前のフォルダに戻る", "", 0, func() {
			currentDir = parentDir
			updateMWFDirectoryList()
		})

		mwfDirList.AddItem("[✔] このフォルダを選択", "", 0, func() {
			targetDir = currentDir
			processFiles()
		})

		// **[追加] 画面を更新 & フォーカス移動**
		app.SetFocus(mwfDirList)
		app.Draw()
	}()
}

// CSVファイルを読み込み、SHA256変換し、出力CSVを作成
func readCSVForOutputFiles() {
	file, err := os.Open(selectedCSV)
	if err != nil {
		log.Fatalf("CSVファイルを開けません: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(transform.NewReader(file, japanese.ShiftJIS.NewDecoder()))
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("CSVの解析に失敗: %v", err)
	}
	var anonymizedRecords [][]string
	outputFiles = []string{}

	for _, record := range records {
		if len(record) > 0 {
			outputFiles = append(outputFiles, record[0])
			if len(record) > 2 {
				patientID := record[2]
				hashedID := sha256Hash(patientID, password)
				patientsTable.Insert(hashedID, record)
				record[2] = hashedID
			}
			anonymizedRecords = append(anonymizedRecords, record[:3])
		} else {
			log.Println("空行があります")
		}
	}
	log.Printf("CSVファイルを読み込みました: %s", selectedCSV)
	saveAnonymizedCSV(anonymizedRecords)
}

// ** SHA256 ハッシュ関数 **
func sha256Hash(patientID, password string) string {
	hash := sha256.Sum256([]byte(patientID + password))
	return hex.EncodeToString(hash[:])
}

// **匿名化したデータをCSVに保存**
func saveAnonymizedCSV(records [][]string) {
	outputPath := filepath.Join(saveDir, filepath.Base(strings.TrimSuffix(selectedCSV, filepath.Ext(selectedCSV))+"_anonymized.csv"))
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("匿名化CSVの作成に失敗: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.WriteAll(records); err != nil {
		log.Fatalf("CSVの書き込みに失敗: %v", err)
	}
	log.Printf("匿名化CSVを出力しました: %s", outputPath)
}

// ** ファイル処理 **
func processFiles() {
	log.Printf("匿名化処理対象フォルダ: %s", targetDir)

	for _, filename := range outputFiles {
		// csvファイル以外はスキップ
		if !strings.HasSuffix(strings.ToLower(filename), ".csv") {
			continue
		}

		filePath := searchFile(targetDir, filename)
		if filePath == "" {
			log.Printf("ファイルが見つかりません: %s", filename)
			continue
		}

		if strings.HasSuffix(strings.ToLower(filePath), ".mwf") {
			wg.Add(1)
			go processFile(filePath)
		}
	}

	wg.Wait()
	app.Stop()
}

// ** ファイル検索 **
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

// **処理したMWFを保存**
func processFile(filePath string) {
	defer wg.Done()

	log.Printf("処理中: %s", filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("読み込み失敗: %v", err)
		return
	}

	anonymizedData, err := mfer.Anonymize(data)
	if err != nil {
		log.Printf("処理失敗: %v", err)
		return
	}

	outputPath := filepath.Join(saveDir, filepath.Base(filePath)) // 保存先を指定
	if err := os.WriteFile(outputPath, anonymizedData, 0666); err != nil {
		log.Printf("書き込み失敗: %v", err)
	}
	log.Printf("処理完了: %s", outputPath)
}
