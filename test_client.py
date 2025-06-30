import websocket
import json
import threading
import time

SERVER_URL = "ws://localhost:8080/ws/control"
UCODE = "123456"  # 你要注册的UCODE

def on_message(ws, message):
    print("收到消息:", message)

def on_error(ws, error):
    print("发生错误:", error)

def on_close(ws, close_status_code, close_msg):
    print("连接关闭")

def on_open(ws):
    print("连接成功，发送注册消息...")
    # 注册UCODE
    register_msg = {
        "type": "register",
        "ucode": UCODE
    }
    ws.send(json.dumps(register_msg))
    print("已发送注册消息:", register_msg)

    # 可选：注册后发送一个ping
    def run():
        time.sleep(1)
        ping_msg = {
            "type": "ping",
            "message": "test ping"
        }
        ws.send(json.dumps(ping_msg))
        print("已发送ping消息")
    threading.Thread(target=run).start()

if __name__ == "__main__":
    ws = websocket.WebSocketApp(
        SERVER_URL,
        on_open=on_open,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close
    )
    ws.run_forever()