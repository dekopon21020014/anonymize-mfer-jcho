# ベースイメージ
FROM golang:1.22.1

# 環境変数設定 (言語と端末設定)
ENV LANG ja_JP.UTF-8
ENV LC_ALL ja_JP.UTF-8
ENV TERM xterm-256color

# 必要なパッケージをインストール (UTF-8サポートのため)
RUN apt-get update && apt-get install -y locales && \
    echo "ja_JP.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen && \
    apt-get clean

# 作業ディレクトリ作成
WORKDIR /app

# モジュールのキャッシュを有効化
COPY go.mod go.sum main.go .env ./
COPY mfer mfer
RUN go mod tidy


# アプリのビルド
RUN go build -o anonymize-mfer

# # エントリーポイント
CMD ["./anonymize-mfer"]