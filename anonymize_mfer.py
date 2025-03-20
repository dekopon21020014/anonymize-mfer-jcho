import sys
import os
import csv
import hashlib
from PyQt6.QtWidgets import (
    QApplication, QWidget, QVBoxLayout, QPushButton, QFileDialog, QMessageBox, QListWidget, QLabel
)
from PyQt6.QtGui import QFont
from PyQt6.QtCore import Qt


class FileProcessorApp(QWidget):
    def __init__(self):
        super().__init__()

        self.selected_csv = None
        self.target_dir = None
        self.output_files = []

        self.initUI()

    def initUI(self):
        self.setWindowTitle("CSVファイルとディレクトリを選択")

        # **フルスクリーン設定（強制）**
        self.showFullScreen()  # 標準的なフルスクリーン
        self.setWindowState(Qt.WindowState.WindowFullScreen)  # macOSなどで強制的に全画面

        layout = QVBoxLayout()

        # **フォント設定（さらに大きく）**
        font = QFont("Arial", 30)  # フォントサイズを30ptに拡大

        # **CSV選択ボタン**
        self.csv_button = QPushButton("📂 CSVファイルを選択", self)
        self.csv_button.setFont(font)
        self.csv_button.setMinimumHeight(80)  # ボタンサイズを大きく
        self.csv_button.clicked.connect(self.select_csv)
        layout.addWidget(self.csv_button)

        self.csv_label = QLabel("未選択", self)
        self.csv_label.setFont(font)
        self.csv_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        layout.addWidget(self.csv_label)

        # **出力ディレクトリ選択ボタン**
        self.dir_button = QPushButton("📁 出力ディレクトリを選択", self)
        self.dir_button.setFont(font)
        self.dir_button.setMinimumHeight(80)
        self.dir_button.clicked.connect(self.select_directory)
        layout.addWidget(self.dir_button)

        self.dir_label = QLabel("未選択", self)
        self.dir_label.setFont(font)
        self.dir_label.setAlignment(Qt.AlignmentFlag.AlignCenter)
        layout.addWidget(self.dir_label)

        # **処理開始ボタン**
        self.process_button = QPushButton("🚀 処理開始", self)
        self.process_button.setFont(font)
        self.process_button.setMinimumHeight(80)
        self.process_button.setEnabled(False)
        self.process_button.clicked.connect(self.process_files)
        layout.addWidget(self.process_button)

        # **処理結果リスト（フォントを大きく）**
        self.result_list = QListWidget()
        self.result_list.setFont(font)
        layout.addWidget(self.result_list)

        # **閉じるボタン（フルスクリーン用）**
        self.close_button = QPushButton("❌ アプリを終了", self)
        self.close_button.setFont(font)
        self.close_button.setMinimumHeight(80)
        self.close_button.clicked.connect(self.close)
        layout.addWidget(self.close_button)

        self.setLayout(layout)

    def select_csv(self):
        file_name, _ = QFileDialog.getOpenFileName(self, "CSVファイルを選択", "", "CSV Files (*.csv)")
        if file_name:
            self.selected_csv = file_name
            self.csv_label.setText(file_name)
            self.read_csv_for_output_files()
            self.check_ready()

    def select_directory(self):
        dir_name = QFileDialog.getExistingDirectory(self, "ディレクトリを選択")
        if dir_name:
            self.target_dir = dir_name
            self.dir_label.setText(dir_name)
            self.check_ready()

    def check_ready(self):
        if self.selected_csv and self.target_dir:
            self.process_button.setEnabled(True)

    def read_csv_for_output_files(self):
        try:
            with open(self.selected_csv, newline='', encoding='shift_jis') as file:
                reader = csv.reader(file)
                records = []
                self.output_files = []

                for row in reader:
                    if len(row) > 0:
                        self.output_files.append(row[0])
                        if len(row) > 2:
                            row[2] = self.sha256_hash(row[2])
                        records.append(row[:3])

                self.save_anonymized_csv(records)
        except Exception as e:
            QMessageBox.critical(self, "エラー", f"CSVの読み込みに失敗しました:\n{str(e)}")

    def save_anonymized_csv(self, records):
        output_path = os.path.splitext(self.selected_csv)[0] + "_anonymized.csv"
        try:
            with open(output_path, 'w', newline='', encoding='utf-8') as file:
                writer = csv.writer(file)
                writer.writerows(records)
            QMessageBox.information(self, "成功", f"匿名化CSVを作成しました:\n{output_path}")
        except Exception as e:
            QMessageBox.critical(self, "エラー", f"匿名化CSVの保存に失敗しました:\n{str(e)}")

    def process_files(self):
        if not self.target_dir or not self.output_files:
            QMessageBox.warning(self, "警告", "ファイルまたはディレクトリが選択されていません。")
            return

        self.result_list.clear()
        found_files = 0

        for filename in self.output_files:
            file_path = self.search_file(self.target_dir, filename)
            if file_path:
                self.result_list.addItem(f"処理中: {filename}")
                self.process_file(file_path)
                found_files += 1
            else:
                self.result_list.addItem(f"見つかりません: {filename}")

        QMessageBox.information(self, "完了", f"{found_files} 個のファイルを処理しました。")

    def search_file(self, root, filename):
        for dirpath, _, filenames in os.walk(root):
            if filename in filenames:
                return os.path.join(dirpath, filename)
        return None

    def process_file(self, file_path):
        try:
            with open(file_path, 'rb') as file:
                data = file.read()

            anonymized_data = self.anonymize_data(data)

            output_dir = os.path.join(os.getcwd(), "anonymized-data")
            os.makedirs(output_dir, exist_ok=True)
            print(output_dir)
            output_path = os.path.join(output_dir, os.path.basename(file_path))
            with open(output_path, 'wb') as file:
                file.write(anonymized_data)

            self.result_list.addItem(f"処理完了: {file_path}")
        except Exception as e:
            self.result_list.addItem(f"処理失敗: {file_path} ({str(e)})")

    def anonymize_data(self, data):
        """ダミーの匿名化処理 (SHA256ハッシュ)"""
        return hashlib.sha256(data).digest()

    def sha256_hash(self, input_str):
        return hashlib.sha256(input_str.encode()).hexdigest()


if __name__ == "__main__":
    app = QApplication(sys.argv)
    window = FileProcessorApp()
    window.show()
    sys.exit(app.exec())
