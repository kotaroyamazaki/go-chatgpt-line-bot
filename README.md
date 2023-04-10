# go-chatgpt-line-bot

## 概要

このプロジェクトは、LINE Messaging API と Firestore を用いた ChatGPT ボットの実装例です。ユーザーが LINE でメッセージを送信すると、ボットがそれに応答します。会話のコンテキストは Firestore に保持され、以前の応答が次の応答に影響を与えることができます。

## 環境変数

以下の環境変数を設定する必要があります。

| 変数名                    | 説明                                                                     |
| ------------------------- | ------------------------------------------------------------------------ |
| LINE_CHANNEL_SECRET       | LINE Developers コンソールから入手した LINE チャネルのシークレット       |
| LINE_CHANNEL_ACCESS_TOKEN | LINE Developers コンソールから入手した LINE チャネルへのアクセストークン |
| OPENAI_API_KEY            | OpenAI API のアクセスキー                                                |
| GCP_PROJECT_ID            | firestore を利用する GCP プロジェクト ID                                 |

# firestore データ構造

```
conversations (collection)
    └─ {userID} (document)
          ├─ messages (array)
          │    ├─ {index} (map)
          │    │    ├─ Role (string)
          │    │    ├─ Content (string)
          │    │    └─ Timestamp (timestamp)
          │    ├─ {index} (map)
          │    │    ├─ Role (string)
          │    │    ├─ Content (string)
          │    │    └─ Timestamp (timestamp)
          │    └─ ...
          └─ expiresAt (timestamp)
```

この構造では、conversations コレクションの下に、ユーザー ID をドキュメント ID として使用しています。各ドキュメントは、messages と expiresAt の 2 つのフィールドを持っています。

messages フィールドは、会話履歴を保持する配列です。配列の各要素は、メッセージの情報を持つマップで構成されており、Role（ユーザーまたは AI）、Content（メッセージ内容）、Timestamp（メッセージが送信された時間）の 3 つのフィールドが含まれています。

expiresAt フィールドは、会話コンテキストの有効期限を表すタイムスタンプです。このフィールドを使用して、古い会話履歴をクリーンアップすることができます。

# デプロイ

以下のコマンドでデプロイできます。

```bash
make deploy
```

# 設定

デプロイ後、発行された HTTP トリガーの URL を、LINE BOT の Messanger API の Webhook URL に設定します。
このとき、アカウント設定の応答メッセージは OFF、Webhook メッセージは ON にしておいてください。
