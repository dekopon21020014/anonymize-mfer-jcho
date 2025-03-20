package model

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

type ECG struct {
	Id        string
	PatientID string
	HashedId  string
	ExportID  string
	Name      string
	Birthtime string
}

func SetupDB(dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS ecgs(
			id TEXT NOT NULL PRIMARY KEY,
			patient_id TEXT NOT NULL,
			hashed_id TEXT NOT NULL,
			export_id TEXT NOT NULL,
			name TEXT,
			birthtime TEXT
		)`,
	}
	for _, query := range queries {
		_, err = db.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// File represents an in-memory file
type File struct {
	Name    string
	Content []byte
}

// ExportPatientsToCSV exports the patients table to an in-memory CSV file
func ExportPatientsToCSV(db *sql.DB) (*File, error) {
	rows, err := db.Query("SELECT * FROM ecgs")
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write([]string{"id", "patient_id", "hashed_id", "export_id", "name", "birthtime"}); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for rows.Next() {
		var id, patientID, hashedID, exportID, name, birthtime string
		if err := rows.Scan(&id, &patientID, &hashedID, &exportID, &name, &birthtime); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if err := writer.Write([]string{id, patientID, hashedID, exportID, name, birthtime}); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error flushing CSV writer: %w", err)
	}

	// Generate a unique filename
	filename := fmt.Sprintf("%s.csv", time.Now().Format("2006-01-02_15-04-05"))

	return &File{
		Name:    filename,
		Content: buf.Bytes(),
	}, nil
}

// Helper function to write the File to an io.Writer
func (f *File) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(f.Content)
	return int64(n), err
}

// 指定されたテーブル全てのエントリを削除
func DeleteAllEntry(db *sql.DB, table string) error {
	// SQL文を作成
	query := fmt.Sprintf("DELETE FROM %s", table)

	// トランザクションを開始
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// SQL文を実行
	_, err = tx.Exec(query)
	if err != nil {
		tx.Rollback()
		return err
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func Put(db *sql.DB, ecg ECG) error {
	// まず、指定されたIDが存在するかを確認
	var existing ECG
	query := `SELECT id, patient_id, hashed_id, export_id, name, birthtime FROM ecgs WHERE id = ?`
	err := db.QueryRow(query, ecg.Id).Scan(
		&existing.Id,
		&existing.PatientID,
		&existing.HashedId,
		&existing.ExportID,
		&existing.Name,
		&existing.Birthtime,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// IDが存在しない場合は、新しいレコードを挿入
			insertQuery := `INSERT INTO ecgs (id, patient_id, hashed_id, export_id, name, birthtime) VALUES (?, ?, ?, ?, ?, ?)`
			_, err := db.Exec(insertQuery, ecg.Id, ecg.PatientID, ecg.HashedId, ecg.ExportID, ecg.Name, ecg.Birthtime)
			if err != nil {
				return fmt.Errorf("failed to insert new ecg: %w", err)
			}
		} else {
			// その他のエラーが発生した場合
			return fmt.Errorf("failed to check existing ecg: %w", err)
		}
	} else {
		log.Println("warning: the ID is already exists")
	}

	return nil
}

func GetHashedIDByExportID(db *sql.DB, exportID string) (string, error) {
	selectQuery := `SELECT hashed_id FROM ecgs WHERE export_id = ?`
	var hashedID string
	err := db.QueryRow(selectQuery, exportID).Scan(&hashedID)
	if err != nil {
		return "", fmt.Errorf("failed to select hashed ID from ecgs: %w", err)
	}
	return hashedID, nil
}
