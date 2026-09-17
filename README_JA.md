# Starbridge AI

Starbridge AI は、自分のドメインとインフラで運用できるマルチモデル AI API ゲートウェイです。統一 API Key、アカウントプール、モデルグループ、スマートスケジューリング、利用量課金、レート制限、決済、管理画面を提供します。

このリポジトリは [Sub2API](https://github.com/Wei-Shaw/sub2api) を基にした Starbridge AI 適応版です。ユーザー向け画面、課金表示、デプロイ文書は Starbridge ブランドで提供しますが、上流プロジェクトの著作権、ライセンス、第三者サービス規約は引き続き有効です。

## 主な機能

- サイト名、ロゴ、サブタイトル、サポート情報、ドキュメント URL のカスタマイズ。
- OpenAI、Anthropic、Gemini、DeepSeek、Grok、Ollama、その他の OpenAI-compatible サービスへの接続。
- API Key、OAuth、Passkey、TOTP、任意の外部 OAuth ログイン。
- アカウントプール、優先度スケジューリング、モデルグループ、フェイルオーバー、同時実行数の制御。
- 残高、サブスクリプション、決済注文、請求書 CSV 出力。
- 顧客向け倍率表示と、管理者向け実コスト表示の分離。
- GPT6 グループでは、設定した低価格モデルで内部要件文書を作成してから、選択した GPT6 モデルが処理を続けます。準備用モデル名は顧客向けモデル名として返しません。
- ログイン前の利用規約、利用ポリシー、対応地域、サービス固有条項の確認。

## クイックスタート

Docker Engine 24+、Docker Compose v2、解決可能なドメインが必要です。

```bash
cd deploy/starbridge
python3 init-env.py --admin-email you@example.com
docker compose up -d --build
```

詳細は [Starbridge デプロイガイド](docs/STARBRIDGE.md) を参照してください。

## コンプライアンスとライセンス

運用者は、利用地域、サーバー所在地、対象ユーザー、上流サービス提供者に適用される法律、プライバシー義務、決済規則、サービス規約を自ら確認し、遵守してください。Starbridge は OpenAI、Anthropic、Google、DeepSeek、その他の上流サービス提供者を代表するものではありません。

本リポジトリは Sub2API 上流プロジェクトの LGPL-3.0 ライセンスと著作権表示を保持します。詳細は [README_UPSTREAM.md](README_UPSTREAM.md) とリポジトリのライセンスファイルを参照してください。
