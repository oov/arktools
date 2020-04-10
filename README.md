# ARKTools

自分用のツール。

## islisten

`/proc/net/tcp` を読むことで該当ポートが Listen されているか調べます。  
`tcp6` は未対応です。

Listen されている場合終了コード `0` が返ります。

### 例: TCP 27020 番が Listen されているか調べる

```
$ arktools islisten tcp 27020
```

## send

RCON サーバーに接続してコマンドを実行します。  
パスワードは必須です。

成功すると標準出力にレスポンスが返り、終了コード `0` で終了します。

### 例: 127.0.0.1:27020 に対して saveworld と doexit を送信する

```
$ arktools send 127.0.0.1 27020 mypassword saveworld && arktools send 127.0.0.1 27020 mypassword doexit
```

## watchdog

サーバーの監視用 HTTP サーバーを開始します。

定期的に RCON サーバーに対して `listplayers` コマンドを送信し、人数を取得します。  
正常に人数が取得できる場合はゲームサーバーが正常に稼働し、ログインができる状態です。

また、第5引数にファイルのパスを指定すると、`/log` へのアクセスで中身を表示します。

### 例: 0.0.0.0:60080 で Listen し、127.0.0.1:27020 の RCON サーバーを監視する

```
$ arktools watchdog 0.0.0.0:60080 127.0.0.1:27020 mypassword /tmp/ark.log
```

`curl http://127.0.0.1:60080/` などで最近の状態が取得でき、`curl http://127.0.0.1:60080/log` で `/tmp/ark.log` の中身が表示されます。
