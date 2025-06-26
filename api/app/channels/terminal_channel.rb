class TerminalChannel < ApplicationCable::Channel
    # クライアントがチャンネルに接続したとき（websocketのリクエストを送ったとき？）に呼ばれる
    def subscribed
        # connection.connection_identifier →connection.rbで定義されている
        stream_from "command_channel_#{connection.connection_identifier}"
    end

    # クライアントからチャンネルが切断された時に呼ばれる
    def unsubscribed
    end

    # クライアントからコマンド実行リクエストを受信したときに呼ばれる
    def execute_command(data)
        # 基本的なバリデーション
        validation_result = validate_command(data["command"])
        if validation_result[:error]
            send_error_response(validation_result[:error])
            return
        end

        # コマンドデータを準備
        command_data = {
            command: data["command"],
            session_id: connection.connection_identifier
        }

        # TerminalExecutorServiceを使用してコマンドを実行
        # このサービスはredisを介して実際のコマンド実行を行う
        begin
            result = TerminalExecutorService.execute(command_data)

            # 実行結果をリクエストを送信したクライアントのみに送信
            ActionCable.server.broadcast(
                "command_channel_#{connection.connection_identifier}",
                result
            )
        rescue => e
            Rails.logger.error "コマンド実行エラー: #{e.message}"
            send_error_response("Command execution failed: #{e.message}")
        end
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
        dangerous_commands = [ "rm", "touch", "sudo", "su", "chmod", "chown" ]
        if dangerous_commands.any? { |cmd| command.include?(cmd) }
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
