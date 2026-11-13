package aura

import (
	"math/rand"
	"time"
)

// PendingEvent is persisted at entry and withheld from public snapshots until its room is cleared.
type DungeonEvent struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Rank            string `json:"rank"`
	Description     string `json:"description"`
	Warning         string `json:"warning"`
	Accepted        bool   `json:"accepted"`
	Phase           int    `json:"phase"`
	RevealAfterRoom int    `json:"reveal_after_room,omitempty"`
}

func plannedEvent(roll, room int) *DungeonEvent {
	event := DungeonEvent{RevealAfterRoom: room, Phase: 1}
	switch {
	case roll < 8:
		event.ID = "false_gate"
		event.Name = "Ложные врата"
		event.Rank = "S"
		event.Description = "За обычной печатью скрыт зал Смотрителя безмолвия. Система исправляет ранг угрозы на S."
		event.Warning = "Три фазы босса, многократно больше здоровья и смертельно опасные ошибки. Время и сложность примеров сохраняются. Можно уйти сейчас с материалами."
	case roll < 20:
		event.ID = "red_gate"
		event.Name = "Красные врата"
		event.Rank = "A"
		event.Description = "Алые руны проступили на стенах. За ними две усиленные комнаты Пепельного узла."
		event.Warning = "Два усиленных боя без святилища. Жар отнимает 2 здоровья за верный удар. Выход после принятия откроется только при победе."
	case roll < 35:
		event.ID = "hidden_shrine"
		event.Name = "Святилище забытых имён"
		event.Rank = "B"
		event.Description = "За тайной стеной пробудилось Эхо первого хранителя. Оно предлагает испытание чистой серии."
		event.Warning = "Один усиленный страж. Ошибка лечит его, верная серия пробивает защиту. Сложность примеров не меняется."
	default:
		return nil
	}
	return &event
}
func planDungeonEvent(session *Session, rnd *rand.Rand) {
	if session.Kind != SessionGate {
		return
	}
	session.PendingEvent = plannedEvent(rnd.Intn(1000), 1+rnd.Intn(2))
}
func revealDungeonEvent(session *Session, now time.Time) bool {
	if session.PendingEvent == nil || session.Event != nil || session.Room != session.PendingEvent.RevealAfterRoom {
		return false
	}
	event := *session.PendingEvent
	event.RevealAfterRoom = 0
	session.PendingEvent = nil
	session.Event = &event
	session.Phase = "event"
	session.ExpiresAt = now.Add(SessionTTL)
	return true
}
func acceptedEvent(session *Session) bool { return session.Event != nil && session.Event.Accepted }
func (s *Service) enterDungeonEvent(session *Session, now time.Time) error {
	if session.Phase != "event" || session.Event == nil || session.Event.Accepted {
		return bad("Событие уже недоступно")
	}
	session.Event.Accepted = true
	session.Event.Phase = 1
	session.Room++
	session.Rooms = session.Room
	if session.Event.ID == "red_gate" {
		session.Rooms++
	}
	session.Combo = 0
	return s.enterRoom(session, "normal", now)
}
func configureEventEnemy(session *Session) {
	if !acceptedEvent(session) {
		return
	}
	power := 18 + session.HeroStats.Str*3
	if session.Empowered {
		session.Empowered = false
	}
	switch session.Event.ID {
	case "false_gate":
		session.BossName = "Смотритель безмолвия"
		session.BossHP = max(280, power*12)
	case "red_gate":
		session.BossName = "Палач Пепельного узла"
		session.BossHP = max(150, power*7)
		if session.Room == session.Rooms {
			session.BossName = "Сердце Пепельного узла"
			session.BossHP = max(220, power*10)
		}
	case "hidden_shrine":
		session.BossName = "Эхо первого хранителя"
		session.BossHP = max(180, power*8)
	}
	session.BossMaxHP = session.BossHP
	session.Event.Phase = 1
}
func eventPhase(session *Session) int {
	if !acceptedEvent(session) || session.BossMaxHP <= 0 {
		return 1
	}
	if session.BossHP*3 <= session.BossMaxHP {
		return 3
	}
	if session.BossHP*3 <= session.BossMaxHP*2 {
		return 2
	}
	return 1
}
func setEventIntent(session *Session) {
	if !acceptedEvent(session) {
		return
	}
	session.Event.Phase = eventPhase(session)
	switch session.Event.ID {
	case "false_gate":
		switch session.Event.Phase {
		case 1:
			session.Intent = "Смотритель: серия из трёх ударов пробивает броню"
		case 2:
			session.Intent = "Приговор: ошибка вызывает усиленную атаку"
		case 3:
			session.Intent = "Безмолвие: ошибка отнимает весь заряд способности"
		}
	case "red_gate":
		session.Intent = "Алый жар: каждый верный удар стоит 2 здоровья; ошибка вызывает мощный удар"
	case "hidden_shrine":
		session.Intent = "Испытание памяти: ошибка лечит стража, серия из трёх ударов пробивает защиту"
	}
}
func eventIncoming(session *Session, base int) int {
	if !acceptedEvent(session) {
		return base
	}
	switch session.Event.ID {
	case "false_gate":
		return max(base, 32+eventPhase(session)*12)
	case "red_gate":
		return max(base, 32+eventPhase(session)*6)
	case "hidden_shrine":
		return max(base, 26+eventPhase(session)*4)
	}
	return base
}
func eventHit(session *Session, damage int) int {
	if !acceptedEvent(session) {
		return damage
	}
	if (session.Event.ID == "false_gate" && eventPhase(session) == 1) || session.Event.ID == "hidden_shrine" {
		if session.Combo%3 != 0 {
			return max(1, damage*2/3)
		}
	}
	return damage
}
func eventMistake(session *Session) {
	if !acceptedEvent(session) {
		return
	}
	switch session.Event.ID {
	case "false_gate":
		if eventPhase(session) == 3 {
			session.Charge = 0
			session.AbilityReady = false
		}
	case "hidden_shrine":
		session.BossHP = min(session.BossMaxHP, session.BossHP+session.BossMaxHP/15)
	}
}
func eventSeals(session *Session) int {
	if !acceptedEvent(session) {
		return 0
	}
	switch session.Event.ID {
	case "false_gate":
		return 3
	case "red_gate":
		return 2
	case "hidden_shrine":
		return 1
	}
	return 0
}
