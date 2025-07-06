# CommandChannel は、WebSocketを通じてクライアントとサーバー間の
# コマンド実行の通信を管理するチャンネル
class CommandChannel < ApplicationCable::Channel
  # クライアントがチャンネルに接続した時に呼ばれる
  # 各クライアントは独自の接続識別子を持つチャンネルを購読
  def subscribed
    # connection.connection_identifier は各クライアントに一意の識別子を割り当て
    # これにより、クライアントごとに独立したチャンネルで通信が可能
    stream_from "command_channel_#{connection.connection_identifier}"
  end

  # クライアントがチャンネルから切断された時に呼ばれる
  def unsubscribed
    # 必要に応じてセッションのクリーンアップなどを実装可能
  end

  # クライアントからコマンド実行リクエストを受信した時に呼ばれる
  # data には { command: "実行するコマンド" } の形式でデータが含まれる
  def execute_command(data)
    # コマンドが空の場合は処理をスキップ
    return unless data["command"].present?

    # CommandExecutorService を使用してコマンドを実行
    # このサービスは Redis を通じて実際のコマンド実行を行う
    result = CommandExecutorService.execute(data["command"])

    # 実行結果を、リクエストを送信したクライアントのみに送信
    # これにより、他のクライアントの結果が混ざることを防止
    ActionCable.server.broadcast(
      "command_channel_#{connection.connection_identifier}",
      result
    )
  end
end
