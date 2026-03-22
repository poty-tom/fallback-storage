# Fallback Storage
画像データなどをストレージサービスに保存する際に、
ベースとするストレージサービスの保存以外に、フォールバック先を考慮した設計を取ることで
継続運用ができる設計を考える。

## 概要
ベースストレージには、`minio` コンテナの代替になる `garage` を採用する。
フォールバック先には確実性を考慮してローカルファイルシステムを想定する。
アプリケーションレイヤで利用しやすい構成を取るため、
フォールバックを考慮する `FallbackStorage` を実装し、内部的に `primary` / `fallback` のストレージクライアントを依存として持つ形をとる。

## 構成
```
fallback-storage
/cmd
    main.go # CLI エントリポイント。環境変数を読み込み、put/get/delete を実行
/internal
    /application
        /storage
            location.go # 保存先を表す Location 定義
            port.go # アプリケーション層で利用する Storage インタフェース
    /injector
        injector.go # Garage / Local / FallbackStorage / Usecase の依存注入
    /storage
        fallback_storage.go # primary 失敗時に fallback へ退避するストレージ実装
        /garage
            garage_client.go # Garage の S3 互換 API を利用する StorageClient 実装
        /local
            local_client.go # ローカルファイルシステムを利用する StorageClient 実装
    /usecase
        storage_access_usecase.go # Put / Get / Delete のユースケース定義
/docker
    /garage
        Dockerfile
        /toml
            garage.toml # Garage コンテナ設定
.env.example # docker-compose と CLI 実行で利用する環境変数テンプレート
```

## コンテナ起動
```bash
cp .env.example .env
docker compose up -d
```

## 動作

### 事前準備
```bash
cp .env.example .env
mkdir -p ./tmp/local-storage
docker compose up -d
set -a; source .env; set +a
```

### 動作確認
動作確認用のテキストファイルを作成する。

```bash
# primary ストレージ用のファイルを作成
printf 'hello fallback storage\n' > sample.txt
# ストレージへの保存
go run ./cmd put ./sample.txt sample.txt

# ストレージからの読み取り
go run ./cmd get sample.txt primary
go run ./cmd get sample.txt fallback

# ストレージからの削除
go run ./cmd delete sample.txt primary
go run ./cmd delete sample.txt fallback
```

使用できるCLI コマンドは以下の通り。
```bash
go run ./cmd put <key> <input-file>
go run ./cmd get <key> <primary|fallback>
go run ./cmd delete <key> <primary|fallback>
```
