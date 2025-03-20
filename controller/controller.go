package controller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sample/mfer"
	"sample/model"
	"sample/xml"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// "github.com/shikidalab/anonymize-ecg/mfer"
	// "github.com/shikidalab/anonymize-ecg/model"
	// "github.com/shikidalab/anonymize-ecg/xml"
)

func GetTop(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome to LP"})
}

const (
	contentTypeZip        = "application/zip"
	contentDispositionFmt = "attachment; filename=%s"
)

var (
	errPasswordMismatch = errors.New("passwords do not match")
	errFileNameFormat   = errors.New("file name format is incorrect")
	errZipCreation      = errors.New("failed to create ZIP file")
	errFileWrite        = errors.New("failed to write file")
)

type File struct {
	Name    string
	Content []byte
}

func AnonymizeECG(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return
	}
	defer conn.Close()

	password, err := validatePassword(conn)
	if err != nil {
		c.String(http.StatusUnauthorized, err.Error())
		log.Println("Error in validate password: ", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, []byte("ok"))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		log.Println("WriteMessage error:", err)
		return
	}

	// バッファ付きのチャネルを使用
	xmlCh := make(chan []File, 10)
	mwfCh := make(chan []File, 10)
	doneCh := make(chan bool)

	// ファイルを受信してチャネルに送る
	go receiveAndBufferFiles(conn, xmlCh, mwfCh)

	// ZIP作成用のバッファとライター
	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	// XMLファイルの処理を完了してからMWFファイルを処理
	go func() {
		// まずXMLファイルを処理
		for xmlFiles := range xmlCh {
			anonymizedFiles, err := processFiles(xmlFiles, password)
			if err != nil {
				log.Println("Error processing XML files:", err)
				continue
			}
			if err := addFilesToZip(zipWriter, anonymizedFiles); err != nil {
				log.Println("Error adding XML files to zip:", err)
			}
		}

		// 次にMWFファイルを処理
		for mwfFiles := range mwfCh {
			anonymizedFiles, err := processFiles(mwfFiles, password)
			if err != nil {
				log.Println("Error processing MWF files:", err)
				continue
			}
			if err := addFilesToZip(zipWriter, anonymizedFiles); err != nil {
				log.Println("Error adding MWF files to zip:", err)
			}
		}

		zipWriter.Close()
		doneCh <- true
	}()

	// 処理完了を待機
	<-doneCh

	sendZipResponse(c, zipBuffer, conn)
	log.Println("The files have been anonymized")
}

func receiveAndBufferFiles(conn *websocket.Conn, xmlCh, mwfCh chan<- []File) {
	ch := make(chan []File)
	go receiveMessage(conn, ch)

	for files := range ch {
		xmlFiles := make([]File, 0)
		mwfFiles := make([]File, 0)

		for _, file := range files {
			if strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
				xmlFiles = append(xmlFiles, file)
			} else if strings.HasSuffix(strings.ToLower(file.Name), ".mwf") {
				mwfFiles = append(mwfFiles, file)
			}
		}

		if len(xmlFiles) > 0 {
			xmlCh <- xmlFiles
		}
		if len(mwfFiles) > 0 {
			mwfCh <- mwfFiles
		}
	}

	close(xmlCh)
	close(mwfCh)
}

// ZIPファイルにファイルを追加するヘルパー関数
func addFilesToZip(zipWriter *zip.Writer, files []File) error {
	for _, file := range files {
		zipFile, err := zipWriter.Create(file.Name)
		if err != nil {
			return fmt.Errorf("%s: %v", errZipCreation, err)
		}
		_, err = zipFile.Write(file.Content)
		if err != nil {
			return fmt.Errorf("%s: %v", errFileWrite, err)
		}
	}
	return nil
}

func validatePassword(conn *websocket.Conn) (string, error) {
	messageType, msg, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("error reading message: %w", err)
	}

	var creds struct {
		Type                 string `json:"type"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"passwordConfirmation"`
	}

	if messageType == websocket.TextMessage {
		err := json.Unmarshal(msg, &creds)
		if err != nil {
			return "", fmt.Errorf("error json.Unmershal: %w", err)
		}

		if creds.Password != creds.PasswordConfirmation {
			return "", fmt.Errorf("error mathing password: %w", errPasswordMismatch)
		}
	}
	return creds.Password, nil
}

func receiveMessage(conn *websocket.Conn, ch chan []File) {
	for {
		// メッセージを受信する
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message in receiveMessage:", err)
			break
		}

		// ZIPファイルをメモリ上で解凍する
		if messageType == websocket.BinaryMessage {
			reader, err := zip.NewReader(bytes.NewReader(msg), int64(len(msg)))
			if err != nil {
				log.Println("Error creating ZIP reader:", err)
				continue
			}
			var files []File
			for _, file := range reader.File {
				// ZIPファイル内のファイルを開く
				rc, err := file.Open()
				if err != nil {
					log.Println("Error opening ZIP file entry:", err)
					continue
				}

				// ファイルの内容を読み込む
				fileContent, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					log.Println("Error reading file content:", err)
					continue
				}

				// ファイル情報を構造体にまとめる
				files = append(files, File{
					Name:    file.Name,
					Content: fileContent,
				})
			}
			ch <- files
		} else if messageType == websocket.TextMessage {
			if bytes.Equal(msg, []byte("end")) {
				log.Println("end of message")
				break
			}
		}
	}
	close(ch)
}

func processFiles(files []File, password string) ([]File, error) {
	var anonymizedFiles []File

	for _, file := range files {
		anonymizedFile, err := processFile(file, password)
		if err != nil {
			log.Println("error in processFile: ", err)
			continue
		}
		if anonymizedFile.Content != nil {
			anonymizedFiles = append(anonymizedFiles, anonymizedFile)
		}
	}
	return anonymizedFiles, nil
}

func processFile(file File, password string) (File, error) {
	fileType := getFileType(file.Name)
	if fileType == "" {
		log.Println("non-mwf and non-xml file, and skipped it")
		return File{}, nil // Skip non-MWF and non-XML files
	}

	exportID, date, err := parseFileName(file.Name)
	if err != nil {
		return File{}, err
	}

	db, err := model.GetDB(os.Getenv("DSN"))
	if err != nil {
		return File{}, err
	}
	defer db.Close()

	var hashedID string
	if fileType == ".xml" { // xmlの時には名前と生年月日を取得する
		ecgID, patientID, name, birthtime, err := xml.GetPersonalInfo(file.Content)
		if err != nil {
			log.Println("getPersonalInfo error: ", err)
		}
		hashedID = hashPatientID(patientID, password)

		err = model.Put(db, model.ECG{
			Id:        ecgID,
			PatientID: patientID,
			HashedId:  hashedID,
			Name:      name,
			Birthtime: birthtime,
			ExportID:  exportID,
		})
		if err != nil {
			return File{}, err
		}
	} else if fileType == ".mwf" {
		hashedID, err = model.GetHashedIDByExportID(db, exportID)
		if err != nil {
			log.Println("GetHashedIDByExportID error: ", err)
		}
	}

	anonymizedData, err := anonymizeData(file.Content, fileType)
	if err != nil {
		return File{}, fmt.Errorf("process file err: %w", err)
	}

	anonymizedFileName := fmt.Sprintf("%s_%s%s", hashedID, date, fileType)
	return File{
		Name:    anonymizedFileName,
		Content: anonymizedData,
	}, nil
}

func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mwf":
		return ".mwf"
	case ".xml":
		return ".xml"
	default:
		return ""
	}
}

func parseFileName(filename string) (string, string, error) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		return "", "", errFileNameFormat
	}
	return parts[0], parts[1], nil
}

func anonymizeData(
	data []byte,
	fileType string,
) ([]byte, error) {
	switch fileType {
	case ".mwf":
		return mfer.Anonymize(data)
	case ".xml":
		return xml.Anonymize(data)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func hashPatientID(patientID, password string) string {
	// 新しいハッシュIDを生成
	newHashedID := sha256.Sum256([]byte(patientID + password))
	hashedIDStr := hex.EncodeToString(newHashedID[:])

	return hashedIDStr
}

func sendZipResponse(c *gin.Context, zipBuffer *bytes.Buffer, conn *websocket.Conn) {

	// 現在の時刻を使用してZIPファイル名を生成
	loc, _ := time.LoadLocation("Asia/Tokyo")
	anonymizedZipFileName := fmt.Sprintf("%s.zip", time.Now().In(loc).Format("2006-01-02_15-04-05"))

	// メタデータを先に送信（例：ファイル名、サイズなど）
	metaData := map[string]string{
		"fileName": anonymizedZipFileName,
		"fileType": contentTypeZip,
	}

	if err := conn.WriteJSON(metaData); err != nil {
		log.Println("error writeJSON: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send metadata"})
		return
	}

	// ZIPファイルデータを送信
	if err := conn.WriteMessage(websocket.BinaryMessage, zipBuffer.Bytes()); err != nil {
		log.Println("error in WriteMessage: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send ZIP file"})
		return
	}
}
