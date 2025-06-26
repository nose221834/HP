# クライアントからきたコマンド実行リクエストをRedisを通して
# 実際の実行サーバに転送し、結果を受け取ってクライアントに返すサービス

class TerminalExecutorService
  # コマンド送信用と結果受信用の2つのチャンネルを使用
  # TODO: あとで環境変数にする
  COMMAND_CHANNEL = "terminal:commands"  # コマンドを送信するチャンネル
  RESULT_CHANNEL = "terminal:results"    # 結果を受信するチャンネル
  TIMEOUT_SECONDS = 10  # コマンド実行のタイムアウト時間（秒）

  # クラスメソッドとして実行を提供
  def self.execute(command)
    new.execute(command)
  end

  # コマンドを実行するためのメインロジック
  # 1. Redisに接続
  # 2. コマンドを送信
  # 3. 結果を待機
  # 4. 結果を返却
  # @param command_data [Hash] 実行するコマンドデータ
  # @return [Hash] 実行結果
  # @raise [StandardError] Redis接続エラーやコマンド実行エラー
  # @example
  #   TerminalExecutorService.execute({ command: "ls -l", session_id: "uuid" })
  #   # => { status: "success", result: "ファイル一覧..." }
  def execute(command_data)
    Rails.logger.info "コマンド実行開始: #{command_data}"

    # コマンドをJson形式に変換
    command_json = {
      command: command_data["command"],
      session_id: command_data["session_id"]
    }.to_json

    # 結果を格納するキューとフラグを初期化
    result_queue = Queue.new
    subscription_active = true

    # 結果を待機するスレッドを開始
    # コードをブロックしないためにスレッドを作成
    result_thread = Thread.new do
      begin
        # 結果チャンネルを購読
        RedisSubscriber.subscribe(RESULT_CHANNEL) do |on|
          # メッセージを受信したら処理
          on.message do |channel, message|
            next unless subscription_active
            begin
              parsed_result = JSON.parse(message)
              Rails.logger.info "結果を受信: #{message}"

              # セッションIDが一致する結果のみを処理
              if command_data["session_id"].nil? || parsed_result["session_id"] == command_data["session_id"]
                result_queue.push(parsed_result)
                subscription_active = false
                RedisSubscriber.unsubscribe
              end
            # パースに失敗した場合はエラーを返す
            rescue JSON::ParserError => e
              Rails.logger.error "JSONパースエラー: #{e.message}, メッセージ: #{message}"
              if subscription_active
                result_queue.push({ status: "error", command: command_data["command"], error: "結果のパースに失敗: #{e.message}" })
                subscription_active = false
                RedisSubscriber.unsubscribe
              end
            end
          end

          # 購読を開始したときに実行される
          on.subscribe do |channel, subscriptions|
            Rails.logger.info "チャンネル購読開始: #{channel} (購読数: #{subscriptions})"
            # 購読開始後にコマンドを送信
            # ここはRedisPublisher（コマンド送信用）を使用する
            RedisPublisher.publish(COMMAND_CHANNEL, command_json)
            Rails.logger.info "コマンドを送信: #{command_json}"
          end

          # 購読を解除したときに実行される
          on.unsubscribe do |channel, subscriptions|
            Rails.logger.info "チャンネル購読解除: #{channel} (購読数: #{subscriptions})"
          end
        # 購読中にエラーが発生した場合はエラーを返す
        rescue => e
          Rails.logger.error "Redis購読エラー: #{e.message}"
          result_queue.push({ status: "error", command: command_data["command"], error: "Redis購読エラー: #{e.message}" })
        # 購読を解除したときに実行される
        ensure
          subscription_active = false
        end
      end
    end

    # 結果を待機（タイムアウト付き）
    begin
      result = result_queue.pop(timeout: TIMEOUT_SECONDS)
      Rails.logger.info "コマンド実行完了: #{result}"
      return result
    rescue ThreadError
      # タイムアウトした場合
      Rails.logger.error "コマンド実行タイムアウト: #{command_data["command"]}"
      subscription_active = false
      RedisSubscriber.unsubscribe if RedisSubscriber.connected?
      return self.class.error_response("Command execution timeout")
    ensure
      # スレッドをクリーンアップ
      result_thread.kill if result_thread.alive?
    end
  end

  private
  # エラーレスポンスの生成
  def self.error_response(message)
    {
      status: "error",
      error: message
    }
  end
end
