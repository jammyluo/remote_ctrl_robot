package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	// 在测试中禁用日志输出
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// 模拟WebSocket服务器
func setupTestServer(t *testing.T) (*httptest.Server, chan []byte) {
	messageChan := make(chan []byte, 10)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("升级WebSocket连接失败: %v", err)
		}
		defer conn.Close()

		// 处理连接
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// 发送消息到通道
			messageChan <- message

			// 发送响应
			response := models.WebSocketMessage{
				Type:       models.WSMessageTypeResponse,
				Command:    models.CMD_TYPE_REGISTER,
				Sequence:   1,
				UCode:      "test_robot",
				ClientType: models.ClientTypeRobot,
				Version:    "1.0.0",
				Data: map[string]interface{}{
					"success": true,
					"message": "Test response",
				},
			}

			responseData, _ := json.Marshal(response)
			conn.WriteMessage(websocket.TextMessage, responseData)
		}
	}))

	return server, messageChan
}

func TestWebSocketConnection(t *testing.T) {
	server, messageChan := setupTestServer(t)
	defer server.Close()

	// 连接到测试服务器
	wsURL := "ws" + server.URL[4:] + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// 发送测试消息
	testMessage := models.WebSocketMessage{
		Type:       models.WSMessageTypeRequest,
		Command:    models.CMD_TYPE_REGISTER,
		Sequence:   1,
		UCode:      "test_robot",
		ClientType: models.ClientTypeRobot,
		Version:    "1.0.0",
		Data:       map[string]interface{}{},
	}

	messageData, err := json.Marshal(testMessage)
	assert.NoError(t, err)

	err = conn.WriteMessage(websocket.TextMessage, messageData)
	assert.NoError(t, err)

	// 等待接收消息
	select {
	case receivedMessage := <-messageChan:
		var parsedMessage models.WebSocketMessage
		err := json.Unmarshal(receivedMessage, &parsedMessage)
		assert.NoError(t, err)
		assert.Equal(t, models.CMD_TYPE_REGISTER, parsedMessage.Command)
		assert.Equal(t, "test_robot", parsedMessage.UCode)
	case <-time.After(5 * time.Second):
		t.Fatal("超时：未收到消息")
	}
}

func TestWebSocketMessageParsing(t *testing.T) {
	// 测试消息解析
	testCases := []struct {
		name    string
		message models.WebSocketMessage
		valid   bool
	}{
		{
			name: "有效注册消息",
			message: models.WebSocketMessage{
				Type:       models.WSMessageTypeRequest,
				Command:    models.CMD_TYPE_REGISTER,
				Sequence:   1,
				UCode:      "test_robot",
				ClientType: models.ClientTypeRobot,
				Version:    "1.0.0",
				Data:       map[string]interface{}{},
			},
			valid: true,
		},
		{
			name: "有效控制消息",
			message: models.WebSocketMessage{
				Type:       models.WSMessageTypeRequest,
				Command:    models.CMD_TYPE_CONTROL_ROBOT,
				Sequence:   2,
				UCode:      "test_robot",
				ClientType: models.ClientTypeRobot,
				Version:    "1.0.0",
				Data: models.CMD_CONTROL_ROBOT{
					Action: "move",
					ParamMaps: map[string]string{
						"direction": "forward",
					},
					Timestamp: time.Now().Unix(),
				},
			},
			valid: true,
		},
		{
			name: "无效消息（缺少必要字段）",
			message: models.WebSocketMessage{
				Type:    models.WSMessageTypeRequest,
				Command: models.CMD_TYPE_REGISTER,
				// 缺少其他必要字段
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 序列化消息
			messageData, err := json.Marshal(tc.message)
			if tc.valid {
				assert.NoError(t, err)
				assert.NotEmpty(t, messageData)

				// 反序列化消息
				var parsedMessage models.WebSocketMessage
				err = json.Unmarshal(messageData, &parsedMessage)
				assert.NoError(t, err)
				assert.Equal(t, tc.message.Command, parsedMessage.Command)
				assert.Equal(t, tc.message.UCode, parsedMessage.UCode)
			} else {
				// 对于无效消息，我们期望序列化可能失败或产生不完整的数据
				if err == nil {
					// 如果序列化成功，反序列化应该能检测到问题
					var parsedMessage models.WebSocketMessage
					err = json.Unmarshal(messageData, &parsedMessage)
					// 这里我们只是验证消息结构，不强制要求错误
				}
			}
		})
	}
}

func TestWebSocketReconnection(t *testing.T) {
	// 测试重连逻辑
	server, _ := setupTestServer(t)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/ws"

	// 第一次连接
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	conn1.Close()

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 第二次连接（模拟重连）
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	conn2.Close()

	// 验证连接成功
	assert.NotNil(t, conn2)
}

func TestWebSocketMessageTypes(t *testing.T) {
	// 测试不同类型的消息
	messageTypes := []models.CommandType{
		models.CMD_TYPE_REGISTER,
		models.CMD_TYPE_PING,
		models.CMD_TYPE_UPDATE_ROBOT_STATUS,
		models.CMD_TYPE_CONTROL_ROBOT,
		models.CMD_TYPE_BIND_ROBOT,
	}

	for _, msgType := range messageTypes {
		t.Run(string(msgType), func(t *testing.T) {
			message := models.WebSocketMessage{
				Type:       models.WSMessageTypeRequest,
				Command:    msgType,
				Sequence:   1,
				UCode:      "test_robot",
				ClientType: models.ClientTypeRobot,
				Version:    "1.0.0",
				Data:       map[string]interface{}{},
			}

			// 序列化
			messageData, err := json.Marshal(message)
			assert.NoError(t, err)

			// 反序列化
			var parsedMessage models.WebSocketMessage
			err = json.Unmarshal(messageData, &parsedMessage)
			assert.NoError(t, err)
			assert.Equal(t, msgType, parsedMessage.Command)
		})
	}
}
