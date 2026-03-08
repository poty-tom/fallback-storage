# Fallback Storage
画像データなどをストレージサービスに保存する際に、
ベースとするストレージサービスの保存以外に、フォールバック先を考慮した設計を取ることで
継続運用ができる設計を考える。

## 概要
ベースストレージには、`minio` コンテナの代替になる `garage` を採用する。
フォールバック先には確実性を考慮してローカルファイルシステムを想定する。
アプリケーションレイヤで利用しやすい構成を取るため、
フォールバックを考慮する `FallbackStorage` を実装し、内部的に`primary`/`fallback`となる`FileStorage` インタフェースを依存として持つ形をとる

## 構成
```
fallback-storage
/internal
    /domain
        /repository
            file_repository.go # アプリケーションレイヤで利用するインタフェース
    /infrastructure
        /storage
            fallback_storage.go # フォールバック込みのストレージサービス
            file_storage.go # 個々のストレージインタフェース
            /garage
                garage.go
            /local
                local.go
```


## コンテナ起動
```bash
cp .env.example .env
docker compose up -d
```