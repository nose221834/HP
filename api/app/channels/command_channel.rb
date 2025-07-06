# CommandChannel は、WebSocketを通じてクライアントとサーバー間の
# コマンド実行の通信を管理するチャンネル
class CommandChannel < ApplicationCable::Channel
  # 危険なコマンドのリスト
  DANGEROUS_COMMANDS = [ "rm", "touch", "sudo", "su", "chmod", "chown" ]

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
  def execute_command(data)
    # dataはすでにHashなのでJSON.parseは不要
    command_data = if data["command"].is_a?(String)
      begin
        JSON.parse(data["command"])
      rescue JSON::ParserError
        { command: data["command"], session_id: nil }
      end
    else
      data["command"] || { command: data["command"], session_id: nil }
    end

    # セッションIDの検証
    client_session_id = command_data["session_id"] rescue nil
    if client_session_id != connection.connection_identifier
        send_error_response("Invalid session ID: Access denied")
        return
    end

    # コマンドのバリデーション
    validation_result = validate_command(command_data["command"])
    if validation_result[:error]
      send_error_response(validation_result[:error])
      return
    end

    # CommandExecutorService を使用してコマンドをRedisに送信して、結果を受信
    result = CommandExecutorService.execute(command_data)

    # 実行結果を、リクエストを送信したクライアントのみに送信
    ActionCable.server.broadcast(
      "command_channel_#{connection.connection_identifier}",
      result
    )
  end

  private
  # コマンドの基本的なバリデーション
  def validate_command(command)
    # nilチェック
    if command.nil?
      return { error: "Command is required" }
    end

    # 空文字チェック
    if command.strip.empty?
      return { error: "Command cannot be empty" }
    end

    # 危険なコマンドのチェック
    if DANGEROUS_COMMANDS.any? { |cmd| command.include?(cmd) }
      return { error: "Dangerous command detected: #{command}" }
    end

    # バリデーション成功
    { error: nil }
  end

  # エラーレスポンスを送信
  def send_error_response(error_message)
    error_result = {
        status: "error",
        error: error_message,
        timestamp: Time.current.iso8601
    }

    ActionCable.server.broadcast(
        "command_channel_#{connection.connection_identifier}",
        error_result
    )
  end
end
