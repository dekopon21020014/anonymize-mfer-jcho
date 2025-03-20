package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sample/mfer"

	"github.com/rivo/tview"
)

var (
	wg  sync.WaitGroup
	app *tview.Application
)

func main() {
	// ログをファイルに出力する
	logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ログファイル作成失敗: %v", err)
	}
	defer logFile.Close()
	defer logFile.Sync()
	log.SetOutput(logFile)

	app = tview.NewApplication()
	list := tview.NewList()
	list.ShowSecondaryText(false).SetBorder(true).SetTitle(" ディレクトリ一覧 ")

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	addDirsToList(app, list, currentDir)

	if err := app.SetRoot(list, true).Run(); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}

	// goroutine の完了を待つ
	wg.Wait()
	log.Println("アプリ終了")
}

// ディレクトリ一覧を表示する関数
func addDirsToList(app *tview.Application, list *tview.List, path string) {
	list.Clear()
	entries, err := os.ReadDir(path)
	if err != nil {
		logMessage("failed to read directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirPath := filepath.Join(path, entry.Name())
			list.AddItem(entry.Name(), "", 0, func() {
				addDirsToList(app, list, dirPath)
			})
		}
	}

	list.AddItem("このディレクトリを選択", "", 0, func() {
		wg.Add(1) // 処理開始前にカウントを増やす
		go func() {
			defer wg.Done() // 処理完了後にカウントを減らす
			processDir(path)
			app.Stop() // TUI を終了
		}()
	})
}

// mwfファイルを探して処理を繰り返す
func processDir(dir string) {
	logMessage("process dir: %s", dir)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".mwf") {
			logMessage("処理中: %s", path)
			data, err := os.ReadFile(path)
			if err != nil {
				logMessage("読み込み失敗: %v", err)
				return nil
			}
			mfer.Anonymize(data)
		}
		return nil
	})

	if err != nil {
		logMessage("ディレクトリ探索中にエラー: %v", err)
	}
}

// ログメッセージを記録する
func logMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
}
