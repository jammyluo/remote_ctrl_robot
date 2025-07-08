package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	mathrand "math/rand"
	"sync"
	"time"

	"remote-ctrl-robot/internal/models"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// GameService 游戏管理器服务
type GameService struct {
	ctx    context.Context
	cancel context.CancelFunc
	mutex  sync.RWMutex

	// 游戏状态
	games map[string]*models.GameState

	// 机器人连接映射
	robotConnections map[string]*websocket.Conn
	connRobots       map[*websocket.Conn]string

	// 游戏配置
	defaultConfig *models.GameConfig

	// 游戏循环
	gameTicker *time.Ticker
}

// NewGameService 创建游戏服务
func NewGameService() *GameService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &GameService{
		ctx:              ctx,
		cancel:           cancel,
		games:            make(map[string]*models.GameState),
		robotConnections: make(map[string]*websocket.Conn),
		connRobots:       make(map[*websocket.Conn]string),
		defaultConfig: &models.GameConfig{
			MaxHealth:     models.DefaultMaxHealth,
			BulletDamage:  models.DefaultBulletDamage,
			BulletRange:   models.DefaultBulletRange,
			BulletSpeed:   models.DefaultBulletSpeed,
			RespawnTime:   models.DefaultRespawnTime,
			GameDuration:  models.DefaultGameDuration,
			MapWidth:      100.0,
			MapHeight:     100.0,
			ShootCooldown: 1.0, // 1秒射击冷却
		},
		gameTicker: time.NewTicker(50 * time.Millisecond), // 50ms游戏循环
	}

	// 启动游戏循环
	go service.gameLoop()

	return service
}

// CreateGame 创建新游戏
func (s *GameService) CreateGame(gameID string) *models.GameState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game := &models.GameState{
		GameID:  gameID,
		Status:  models.GameStatusWaiting,
		Robots:  make(map[string]*models.GameRobot),
		Bullets: make([]*models.GameBullet, 0),
		Config:  s.defaultConfig,
		Statistics: &models.GameStatistics{
			RobotStats: make(map[string]*models.RobotStats),
			GameEvents: make([]*models.GameEvent, 0),
		},
	}

	s.games[gameID] = game
	log.Info().Str("game_id", gameID).Msg("Game created")
	return game
}

// JoinGame 加入游戏
func (s *GameService) JoinGame(gameID, ucode, name string, conn *websocket.Conn) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if game.Status != models.GameStatusWaiting {
		return fmt.Errorf("game %s is not in waiting status", gameID)
	}

	// 检查机器人是否已在游戏中
	if _, exists := game.Robots[ucode]; exists {
		return fmt.Errorf("robot %s already in game", ucode)
	}

	// 创建游戏机器人
	robot := &models.GameRobot{
		UCode:     ucode,
		Name:      name,
		Health:    game.Config.MaxHealth,
		MaxHealth: game.Config.MaxHealth,
		Position: models.Position{
			X: randFloat(-game.Config.MapWidth/2, game.Config.MapWidth/2),
			Y: randFloat(-game.Config.MapHeight/2, game.Config.MapHeight/2),
			Z: 0,
		},
		Direction:   randFloat(0, 2*math.Pi),
		IsAlive:     true,
		LastShot:    time.Now().Add(-time.Duration(game.Config.ShootCooldown) * time.Second),
		RespawnTime: time.Now(),
		Score:       0,
		Kills:       0,
		Deaths:      0,
		ShotsFired:  0,
		ShotsHit:    0,
	}

	game.Robots[ucode] = robot
	s.robotConnections[ucode] = conn
	s.connRobots[conn] = ucode

	// 创建机器人统计
	game.Statistics.RobotStats[ucode] = &models.RobotStats{
		UCode:      ucode,
		Score:      0,
		Kills:      0,
		Deaths:     0,
		ShotsFired: 0,
		ShotsHit:   0,
		Accuracy:   0.0,
		KDRatio:    0.0,
	}

	// 添加游戏事件
	event := &models.GameEvent{
		Type:         "join",
		Timestamp:    time.Now(),
		ShooterUCode: ucode,
		Message:      fmt.Sprintf("Robot %s joined the game", name),
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	log.Info().Str("game_id", gameID).Str("ucode", ucode).Str("name", name).Msg("Robot joined game")

	// 广播游戏状态
	s.broadcastGameState(gameID)

	return nil
}

// LeaveGame 离开游戏
func (s *GameService) LeaveGame(gameID, ucode string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if _, exists := game.Robots[ucode]; !exists {
		return fmt.Errorf("robot %s not in game", ucode)
	}

	// 移除机器人
	delete(game.Robots, ucode)
	delete(game.Statistics.RobotStats, ucode)

	// 移除连接映射
	if conn, exists := s.robotConnections[ucode]; exists {
		delete(s.connRobots, conn)
		delete(s.robotConnections, ucode)
	}

	// 添加游戏事件
	event := &models.GameEvent{
		Type:         "leave",
		Timestamp:    time.Now(),
		ShooterUCode: ucode,
		Message:      fmt.Sprintf("Robot %s left the game", game.Robots[ucode].Name),
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	log.Info().Str("game_id", gameID).Str("ucode", ucode).Msg("Robot left game")

	// 广播游戏状态
	s.broadcastGameState(gameID)

	return nil
}

// StartGame 开始游戏
func (s *GameService) StartGame(gameID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if game.Status != models.GameStatusWaiting {
		return fmt.Errorf("game %s is not in waiting status", gameID)
	}

	if len(game.Robots) < 2 {
		return fmt.Errorf("need at least 2 robots to start game")
	}

	game.Status = models.GameStatusPlaying
	game.StartTime = time.Now()
	game.EndTime = game.StartTime.Add(time.Duration(game.Config.GameDuration) * time.Second)

	// 添加游戏事件
	event := &models.GameEvent{
		Type:      "game_start",
		Timestamp: time.Now(),
		Message:   "Game started",
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	log.Info().Str("game_id", gameID).Msg("Game started")

	// 广播游戏状态
	s.broadcastGameState(gameID)

	return nil
}

// StopGame 停止游戏
func (s *GameService) StopGame(gameID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	game.Status = models.GameStatusFinished
	game.EndTime = time.Now()

	// 计算获胜者
	s.calculateWinner(game)

	// 添加游戏事件
	event := &models.GameEvent{
		Type:      "game_end",
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Game ended. Winner: %s", game.Winner),
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	log.Info().Str("game_id", gameID).Str("winner", game.Winner).Msg("Game stopped")

	// 广播游戏状态
	s.broadcastGameState(gameID)

	return nil
}

// ProcessShot 处理射击
func (s *GameService) ProcessShot(gameID, shooterUCode string, targetX, targetY, targetZ float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if game.Status != models.GameStatusPlaying {
		return fmt.Errorf("game %s is not in playing status", gameID)
	}

	shooter, exists := game.Robots[shooterUCode]
	if !exists {
		return fmt.Errorf("shooter %s not found", shooterUCode)
	}

	if !shooter.IsAlive {
		return fmt.Errorf("shooter %s is not alive", shooterUCode)
	}

	// 检查射击冷却
	if time.Since(shooter.LastShot) < time.Duration(game.Config.ShootCooldown)*time.Second {
		return fmt.Errorf("shooter %s is in cooldown", shooterUCode)
	}

	// 创建子弹
	bulletID := generateID()
	targetPos := models.Position{X: targetX, Y: targetY, Z: targetZ}

	// 计算射击方向
	direction := s.calculateDirection(shooter.Position, targetPos)

	bullet := &models.GameBullet{
		ID:           bulletID,
		ShooterUCode: shooterUCode,
		StartPos:     shooter.Position,
		CurrentPos:   shooter.Position,
		Direction:    direction,
		Speed:        game.Config.BulletSpeed,
		Damage:       game.Config.BulletDamage,
		Range:        game.Config.BulletRange,
		Created:      time.Now(),
		IsActive:     true,
	}

	game.Bullets = append(game.Bullets, bullet)
	shooter.LastShot = time.Now()
	shooter.ShotsFired++
	game.Statistics.TotalShots++

	// 更新统计
	if stats, exists := game.Statistics.RobotStats[shooterUCode]; exists {
		stats.ShotsFired++
	}

	// 添加射击事件
	event := &models.GameEvent{
		Type:         "shot",
		Timestamp:    time.Now(),
		ShooterUCode: shooterUCode,
		Position:     shooter.Position,
		Message:      fmt.Sprintf("Robot %s fired a shot", shooter.Name),
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	log.Debug().Str("game_id", gameID).Str("shooter", shooterUCode).Msg("Shot processed")

	return nil
}

// ProcessMove 处理移动
func (s *GameService) ProcessMove(gameID, ucode string, position models.Position, direction float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	game, exists := s.games[gameID]
	if !exists {
		return fmt.Errorf("game %s not found", gameID)
	}

	if game.Status != models.GameStatusPlaying {
		return fmt.Errorf("game %s is not in playing status", gameID)
	}

	robot, exists := game.Robots[ucode]
	if !exists {
		return fmt.Errorf("robot %s not found", ucode)
	}

	if !robot.IsAlive {
		return fmt.Errorf("robot %s is not alive", ucode)
	}

	// 检查边界
	if math.Abs(position.X) > game.Config.MapWidth/2 || math.Abs(position.Y) > game.Config.MapHeight/2 {
		return fmt.Errorf("position out of bounds")
	}

	robot.Position = position
	robot.Direction = direction

	return nil
}

// GetGameState 获取游戏状态
func (s *GameService) GetGameState(gameID string) (*models.GameState, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	game, exists := s.games[gameID]
	if !exists {
		return nil, fmt.Errorf("game %s not found", gameID)
	}

	return game, nil
}

// GetRobotInGame 获取机器人在游戏中的状态
func (s *GameService) GetRobotInGame(gameID, ucode string) (*models.GameRobot, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	game, exists := s.games[gameID]
	if !exists {
		return nil, fmt.Errorf("game %s not found", gameID)
	}

	robot, exists := game.Robots[ucode]
	if !exists {
		return nil, fmt.Errorf("robot %s not found in game", ucode)
	}

	return robot, nil
}

// RemoveRobotConnection 移除机器人连接
func (s *GameService) RemoveRobotConnection(conn *websocket.Conn) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if ucode, exists := s.connRobots[conn]; exists {
		delete(s.robotConnections, ucode)
		delete(s.connRobots, conn)

		// 从所有游戏中移除该机器人
		for gameID, game := range s.games {
			if _, exists := game.Robots[ucode]; exists {
				s.LeaveGame(gameID, ucode)
			}
		}
	}
}

// 游戏主循环
func (s *GameService) gameLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.gameTicker.C:
			s.updateGames()
		}
	}
}

// 更新所有游戏状态
func (s *GameService) updateGames() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for gameID, game := range s.games {
		if game.Status == models.GameStatusPlaying {
			s.updateGame(gameID, game)
		}
	}
}

// 更新单个游戏
func (s *GameService) updateGame(gameID string, game *models.GameState) {
	// 检查游戏是否结束
	if time.Now().After(game.EndTime) {
		s.calculateWinner(game)
		game.Status = models.GameStatusFinished

		event := &models.GameEvent{
			Type:      "game_end",
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Game ended. Winner: %s", game.Winner),
		}
		game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

		s.broadcastGameState(gameID)
		return
	}

	// 更新子弹
	s.updateBullets(game)

	// 检查复活
	s.checkRespawns(game)

	// 广播游戏状态
	s.broadcastGameState(gameID)
}

// 更新子弹
func (s *GameService) updateBullets(game *models.GameState) {
	activeBullets := make([]*models.GameBullet, 0)

	for _, bullet := range game.Bullets {
		if !bullet.IsActive {
			continue
		}

		// 移动子弹
		bullet.CurrentPos.X += bullet.Direction.X * bullet.Speed * 0.05 // 50ms
		bullet.CurrentPos.Y += bullet.Direction.Y * bullet.Speed * 0.05
		bullet.CurrentPos.Z += bullet.Direction.Z * bullet.Speed * 0.05

		// 检查射程
		distance := s.calculateDistance(bullet.StartPos, bullet.CurrentPos)
		if distance > bullet.Range {
			bullet.IsActive = false
			continue
		}

		// 检查碰撞
		hitRobot := s.checkBulletCollision(bullet, game)
		if hitRobot != nil {
			bullet.IsActive = false
			s.applyDamage(hitRobot, bullet.Damage, bullet.ShooterUCode, game)
		}

		if bullet.IsActive {
			activeBullets = append(activeBullets, bullet)
		}
	}

	game.Bullets = activeBullets
}

// 检查子弹碰撞
func (s *GameService) checkBulletCollision(bullet *models.GameBullet, game *models.GameState) *models.GameRobot {
	for _, robot := range game.Robots {
		if !robot.IsAlive || robot.UCode == bullet.ShooterUCode {
			continue
		}

		distance := s.calculateDistance(bullet.CurrentPos, robot.Position)
		if distance <= 2.0 { // 碰撞半径
			return robot
		}
	}

	return nil
}

// 应用伤害
func (s *GameService) applyDamage(robot *models.GameRobot, damage int, shooterUCode string, game *models.GameState) {
	robot.Health = max(0, robot.Health-damage)

	// 更新射击者统计
	if shooter, exists := game.Robots[shooterUCode]; exists {
		shooter.ShotsHit++
		if stats, exists := game.Statistics.RobotStats[shooterUCode]; exists {
			stats.ShotsHit++
		}
	}

	game.Statistics.TotalHits++

	// 添加命中事件
	event := &models.GameEvent{
		Type:         "hit",
		Timestamp:    time.Now(),
		ShooterUCode: shooterUCode,
		TargetUCode:  robot.UCode,
		Damage:       damage,
		Position:     robot.Position,
		Message: fmt.Sprintf("Robot %s hit %s for %d damage",
			game.Robots[shooterUCode].Name, robot.Name, damage),
	}
	game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)

	// 检查是否死亡
	if robot.Health <= 0 {
		robot.IsAlive = false
		robot.Deaths++
		robot.RespawnTime = time.Now().Add(time.Duration(game.Config.RespawnTime) * time.Second)

		// 更新击杀者统计
		if shooter, exists := game.Robots[shooterUCode]; exists {
			shooter.Kills++
			shooter.Score += 10
			if stats, exists := game.Statistics.RobotStats[shooterUCode]; exists {
				stats.Kills++
				stats.Score += 10
			}
		}

		game.Statistics.TotalKills++
		game.Statistics.TotalDeaths++

		// 添加击杀事件
		killEvent := &models.GameEvent{
			Type:         "kill",
			Timestamp:    time.Now(),
			ShooterUCode: shooterUCode,
			TargetUCode:  robot.UCode,
			Position:     robot.Position,
			Message: fmt.Sprintf("Robot %s killed %s",
				game.Robots[shooterUCode].Name, robot.Name),
		}
		game.Statistics.GameEvents = append(game.Statistics.GameEvents, killEvent)
	}
}

// 检查复活
func (s *GameService) checkRespawns(game *models.GameState) {
	for _, robot := range game.Robots {
		if !robot.IsAlive && time.Now().After(robot.RespawnTime) {
			robot.IsAlive = true
			robot.Health = game.Config.MaxHealth
			robot.Position = models.Position{
				X: randFloat(-game.Config.MapWidth/2, game.Config.MapWidth/2),
				Y: randFloat(-game.Config.MapHeight/2, game.Config.MapHeight/2),
				Z: 0,
			}
			robot.Direction = randFloat(0, 2*math.Pi)

			// 添加复活事件
			event := &models.GameEvent{
				Type:         "respawn",
				Timestamp:    time.Now(),
				ShooterUCode: robot.UCode,
				Position:     robot.Position,
				Message:      fmt.Sprintf("Robot %s respawned", robot.Name),
			}
			game.Statistics.GameEvents = append(game.Statistics.GameEvents, event)
		}
	}
}

// 计算获胜者
func (s *GameService) calculateWinner(game *models.GameState) {
	var winner *models.GameRobot
	maxScore := -1

	for _, robot := range game.Robots {
		if robot.Score > maxScore {
			maxScore = robot.Score
			winner = robot
		}
	}

	if winner != nil {
		game.Winner = winner.UCode
	}
}

// 广播游戏状态
func (s *GameService) broadcastGameState(gameID string) {
	game, exists := s.games[gameID]
	if !exists {
		return
	}

	// 为每个机器人发送个性化状态
	for ucode, robot := range game.Robots {
		if conn, exists := s.robotConnections[ucode]; exists {
			response := models.WebSocketMessage{
				Type:     models.WSMessageTypeResponse,
				Command:  models.CMD_TYPE_GAME_STATUS,
				Sequence: time.Now().UnixNano(),
				UCode:    ucode,
				Data: models.CMD_GAME_STATUS_RESPONSE{
					GameState: game,
					MyRobot:   robot,
				},
			}

			if err := conn.WriteJSON(response); err != nil {
				log.Error().Err(err).Str("ucode", ucode).Msg("Failed to send game state")
			}
		}
	}
}

// 工具函数
func (s *GameService) calculateDistance(pos1, pos2 models.Position) float64 {
	dx := pos1.X - pos2.X
	dy := pos1.Y - pos2.Y
	dz := pos1.Z - pos2.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (s *GameService) calculateDirection(from, to models.Position) models.Position {
	dx := to.X - from.X
	dy := to.Y - from.Y
	dz := to.Z - from.Z

	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if distance == 0 {
		return models.Position{X: 0, Y: 0, Z: 0}
	}

	return models.Position{
		X: dx / distance,
		Y: dy / distance,
		Z: dz / distance,
	}
}

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func randFloat(min, max float64) float64 {
	return min + mathrand.Float64()*(max-min)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Shutdown 关闭服务
func (s *GameService) Shutdown() {
	s.cancel()
	s.gameTicker.Stop()
}
