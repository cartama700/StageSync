# StageSync

> **リズムゲームバックエンド サーバーエンジニアの日常業務シミュレーション** — 株式会社 Colorful Palette サーバサイドエンジニア募集向けポートフォリオ。

**言語 / 언어**: [日本語 (本ファイル)](./README.ja.md) · [한국어](./README.md)
**SSOT**: [`docs/MISSION.md`](./docs/MISSION.md) · [`docs/PLAN.md`](./docs/PLAN.md) (韓国語)

---

## 一行ミッション

**株式会社 Colorful Palette サーバサイドエンジニア [BA-09-04a](https://hrmos.co/pages/colorfulpalette/jobs/BA-09-04a) の日常業務を Go でシミュレートしたポートフォリオ。**

リズムゲームバックエンド — **プロフィール · ガチャ · イベント · ランキング · メール** — を Spring Boot 相当の Go 構造 (handler-service-repository 3 層) で実装し、求人明記スタック (Aurora MySQL · Spanner · Redis · GKE · Terraform · Locust) をそのまま整合。

---

## 作者について (正直に)

本プロジェクトは **「職人の代表作」** ではなく **「成長ポートフォリオ」** です。

- **Go · Java 今回が初の実務挑戦** (C#/.NET 実務数年)
- **日本語 N2 ~ N3** (求人 N1 必須 · エージェント経由でコンタクト成立、ポートフォリオが決め手)
- **AI ツール (Claude Code) 積極活用**
- 証明したいこと: **学習速度** · **技術的深さ** · **日本業界慣行への整合意志**

---

## ステータス

- **ボーナス軸 完了** (Phase 0, A, B 3/3): 骨格 · WebSocket Room · AOI トグル
- **メイン軸 v0.1 準備中**: REST + clean architecture + MySQL + テスト CI

詳細 Phase ロードマップ: [`docs/PLAN.md`](./docs/PLAN.md)

---

## クイックスタート

```bash
make tidy && make proto && make build
make run                                     # サーバ起動 (:5050)
go run ./cmd/bots -player=p1 -tick=200       # ボーナス軸: 負荷ボット
curl http://localhost:5050/api/metrics
# → {"connectedPlayers":1,"optimized":false,"tps":0}
```

---

## 戦略の再定義 (2026-04-18)

初期は「C# Aiming PoC をリアルタイム中心の 5 軸フレームで Go 移植」でしたが、**実際の求人 BA-09-04a はリアルタイム通信が Diarkis 外部ミドルウェア担当で、この職種の日常業務は REST API · DB · 非同期バッチ · 運用中心**。

そのため大前提を REST-first に再定義:
- Phase 0 · A · B (旧 0·1·2) はボーナス軸として保持
- 新メイン軸 Phase 1-18: REST + MySQL + ゲームドメイン + 運用 + デプロイ + 仕上げ

---

## 詳細ドキュメント

- [**README.md**](./README.md) (韓国語メイン、最新)
- [**docs/MISSION.md**](./docs/MISSION.md) (プロジェクト SSOT · 求人マッピング · 5 軸フレーム)
- [**docs/PLAN.md**](./docs/PLAN.md) (Phase 0-18 ロードマップ · 依存関係 · 学習トラッカー)

**本ファイルは Phase 18 で完全版に置き換え予定。** 現在は作者の日本語レベル N2~N3 を考慮し、最小限の概要にとどめています。

---

## ライセンス

個人ポートフォリオ用、商用再利用制限。
